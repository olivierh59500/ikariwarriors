package game

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
	gameassets "github.com/olivierh59500/ikariwarriors/assets"
)

const (
	mapColumns       = 13
	mapRows          = 938
	tileSize         = 16
	secondaryMapRows = 300
	bobSize          = 16
	bobBytes         = 160
	actorBobWidth    = 32
	actorBobHeight   = 16
	actorBobBytes    = 320
	overlayBobHeight = 8
	overlayBobBytes  = 160
	vehicleBobHeight = 24
	vehicleBobBytes  = 480
	levelChunkHeight = 1024
)

type tileSet struct {
	images []*ebiten.Image
	cpu    []image.Image
}

type compositeSize struct{ rows, columns int }

type resources struct {
	composites   [19]compositeSize
	title        *ebiten.Image
	titleActors  *ebiten.Image
	intro        *ebiten.Image
	font         *ebiten.Image
	fuelGauge    *ebiten.Image
	fuelEmpty    *ebiten.Image
	colors       color.Palette
	tiles        [2]tileSet
	level        [mapRows][mapColumns]uint16
	levelCodes   [mapRows][mapColumns]byte
	passable     [2][256]bool
	effects      []*ebiten.Image
	players      [2][]*ebiten.Image
	playerParts  []*ebiten.Image
	enemies      []*ebiten.Image
	enemyParts   []*ebiten.Image
	vehicles     [3][]*ebiten.Image
	levelChunks  []*ebiten.Image
	events       []levelEvent
	endingFrames [2][12]byte
	bobData      []byte
	bobImages    map[bobKey]*ebiten.Image
	mine         *ebiten.Image
}

type bobKey struct {
	offset int
	height int
}

func loadResources() (*resources, error) {
	paletteData, err := gameassets.Original.ReadFile("original/palette.bin")
	if err != nil {
		return nil, err
	}
	fontData, err := gameassets.Original.ReadFile("original/font.bin")
	if err != nil {
		return nil, err
	}
	levelData, err := gameassets.Original.ReadFile("original/level.bin")
	if err != nil {
		return nil, err
	}
	bobData, err := gameassets.Original.ReadFile("original/bobs-primary.bin")
	if err != nil {
		return nil, err
	}
	if len(paletteData) != 32 || len(levelData) != mapRows*mapColumns*2 || len(bobData)%bobBytes != 0 {
		return nil, fmt.Errorf("recovered Ikari Warriors data has an invalid size")
	}

	palette := make(color.Palette, 16)
	for index := range palette {
		value := binary.BigEndian.Uint16(paletteData[index*2:]) * 2
		palette[index] = color.NRGBA{
			R: uint8(value>>8&15) * 17,
			G: uint8(value>>4&15) * 17,
			B: uint8(value&15) * 17,
			A: 255,
		}
	}

	result := &resources{colors: palette}
	mineData, err := gameassets.Original.ReadFile("original/bobs-mine.bin")
	if err != nil {
		return nil, err
	}
	if len(mineData) != bobBytes {
		return nil, fmt.Errorf("recovered mine graphic has an invalid size")
	}
	result.mine = decodeBob(mineData, palette)
	result.bobData, err = gameassets.Original.ReadFile("original/bobs-complete.bin")
	if err != nil {
		return nil, err
	}
	result.bobImages = make(map[bobKey]*ebiten.Image)
	endingData, err := gameassets.Original.ReadFile("original/ending-frames.bin")
	if err != nil {
		return nil, err
	}
	if len(endingData) != 24 {
		return nil, fmt.Errorf("recovered ending animation has an invalid size")
	}
	copy(result.endingFrames[0][:], endingData[:12])
	copy(result.endingFrames[1][:], endingData[12:])
	events, err := gameassets.Original.ReadFile("original/level-events.bin")
	if err != nil {
		return nil, err
	}
	if len(events)%6 != 0 {
		return nil, fmt.Errorf("recovered event table has an invalid size")
	}
	for offset := 0; offset < len(events); offset += 6 {
		result.events = append(result.events, levelEvent{
			progress: int(binary.BigEndian.Uint16(events[offset:])),
			x:        int(events[offset+2]), count: int(events[offset+3]),
			kind: int(events[offset+4]), spacing: int(events[offset+5]),
		})
	}
	codes, err := gameassets.Original.ReadFile("original/level-codes.bin")
	if err != nil {
		return nil, err
	}
	if len(codes) != mapRows*mapColumns {
		return nil, fmt.Errorf("recovered collision data has an invalid size")
	}
	for row := range mapRows {
		copy(result.levelCodes[row][:], codes[row*mapColumns:(row+1)*mapColumns])
	}
	for bank, name := range []string{"terrain-passable.bin", "terrain-passable-secondary.bin"} {
		passable, readErr := gameassets.Original.ReadFile("original/" + name)
		if readErr != nil {
			return nil, readErr
		}
		if len(passable) != 98 {
			return nil, fmt.Errorf("recovered passable table has an invalid size")
		}
		for _, code := range passable {
			result.passable[bank][code] = true
		}
	}
	dimensions, err := gameassets.Original.ReadFile("original/composite-dimensions.bin")
	if err != nil {
		return nil, err
	}
	if len(dimensions) != len(result.composites)*4 {
		return nil, fmt.Errorf("recovered composite dimensions have an invalid size")
	}
	for i := range result.composites {
		result.composites[i] = compositeSize{int(binary.BigEndian.Uint16(dimensions[i*4:])) + 1, int(binary.BigEndian.Uint16(dimensions[i*4+2:])) + 1}
	}
	result.title, err = loadPNG("original/title.png")
	if err != nil {
		return nil, err
	}
	result.titleActors, err = loadPNG("original/title-actors.png")
	if err != nil {
		return nil, err
	}
	result.intro, err = loadPNG("original/intro.png")
	if err != nil {
		return nil, err
	}
	result.font = decodeFont(fontData, palette)
	result.fuelGauge, result.fuelEmpty = decodeFuelGauge(fontData, palette)
	for row := range mapRows {
		for column := range mapColumns {
			offset := (row*mapColumns + column) * 2
			result.level[row][column] = binary.BigEndian.Uint16(levelData[offset:])
		}
	}
	for bank, name := range []string{"tiles-primary.bin", "tiles-secondary.bin"} {
		data, readErr := gameassets.Original.ReadFile("original/" + name)
		if readErr != nil {
			return nil, readErr
		}
		result.tiles[bank], err = decodeTiles(data, palette)
		if err != nil {
			return nil, err
		}
	}
	result.effects = decodeBobBank(bobData, palette)
	playerData, readErr := gameassets.Original.ReadFile("original/bobs-title-a.bin")
	if readErr != nil {
		return nil, readErr
	}
	playerFrames, err := decodeActorBobBank(playerData, palette)
	if err != nil {
		return nil, err
	}
	if len(playerFrames) != 44 {
		return nil, fmt.Errorf("recovered player bank has %d frames", len(playerFrames))
	}
	result.players[0] = playerFrames[:22]
	result.players[1] = playerFrames[22:]
	playerPartData, readErr := gameassets.Original.ReadFile("original/bobs-title-b.bin")
	if readErr != nil {
		return nil, readErr
	}
	result.playerParts, err = decodeOverlayBobBank(playerPartData, palette)
	if err != nil {
		return nil, err
	}
	enemyData, readErr := gameassets.Original.ReadFile("original/bobs-enemies-a.bin")
	if readErr != nil {
		return nil, readErr
	}
	result.enemies, err = decodeActorBobBank(enemyData, palette)
	if err != nil {
		return nil, err
	}
	enemyPartData, readErr := gameassets.Original.ReadFile("original/bobs-enemies-b.bin")
	if readErr != nil {
		return nil, readErr
	}
	result.enemyParts, err = decodeOverlayBobBank(enemyPartData, palette)
	if err != nil {
		return nil, err
	}
	for bank, name := range []string{"bobs-tank-enemy.bin", "bobs-tank-player1.bin", "bobs-tank-player2.bin"} {
		data, readErr := gameassets.Original.ReadFile("original/" + name)
		if readErr != nil {
			return nil, readErr
		}
		result.vehicles[bank], err = decodeVehicleBobBank(data, palette)
		if err != nil {
			return nil, err
		}
	}
	result.levelChunks = buildLevelChunks(result)
	return result, nil
}

// bob decodes a recovered graphic by its byte offset in the complete art bank.
// Height belongs to the graphic format, rather than its gameplay category.
func (r *resources) bob(offset, height int) *ebiten.Image {
	key := bobKey{offset, height}
	if cached := r.bobImages[key]; cached != nil {
		return cached
	}
	end := offset + height*actorBobWidth/8*5
	if offset < 0 || end > len(r.bobData) {
		return nil
	}
	decoded := decodeWideBob(r.bobData[offset:end], height, r.colors)
	r.bobImages[key] = decoded
	return decoded
}

func loadPNG(name string) (*ebiten.Image, error) {
	file, err := gameassets.Original.Open(name)
	if err != nil {
		return nil, err
	}
	decoded, err := png.Decode(file)
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return ebiten.NewImageFromImage(decoded), nil
}

func decodeTiles(data []byte, palette color.Palette) (tileSet, error) {
	if len(data)%128 != 0 {
		return tileSet{}, fmt.Errorf("recovered tile bank has an invalid size")
	}
	result := tileSet{
		images: make([]*ebiten.Image, len(data)/128),
		cpu:    make([]image.Image, len(data)/128),
	}
	for tile := range result.images {
		decoded := image.NewPaletted(image.Rect(0, 0, tileSize, tileSize), palette)
		base := tile * 128
		for y := range tileSize {
			for x := range tileSize {
				value := uint8(0)
				for plane := range 4 {
					if data[base+plane*32+y*2+x/8]&(1<<uint(7-x%8)) != 0 {
						value |= 1 << plane
					}
				}
				decoded.SetColorIndex(x, y, value)
			}
		}
		result.cpu[tile] = decoded
		result.images[tile] = ebiten.NewImageFromImage(decoded)
	}
	return result, nil
}

func buildLevelChunks(source *resources) []*ebiten.Image {
	chunks := make([]*ebiten.Image, (levelHeight+levelChunkHeight-1)/levelChunkHeight)
	for chunk := range chunks {
		top := chunk * levelChunkHeight
		height := min(levelChunkHeight, levelHeight-top)
		decoded := image.NewRGBA(image.Rect(0, 0, playfieldWidth, height))
		firstRow := top / tileSize
		lastRow := min(mapRows, (top+height+tileSize-1)/tileSize)
		for row := firstRow; row < lastRow; row++ {
			bank := 0
			if row < secondaryMapRows {
				bank = 1
			}
			for column := range mapColumns {
				tile := int(source.level[row][column])
				if tile >= len(source.tiles[bank].cpu) {
					continue
				}
				destination := image.Rect(column*tileSize, row*tileSize-top, (column+1)*tileSize, row*tileSize-top+tileSize)
				draw.Draw(decoded, destination, source.tiles[bank].cpu[tile], image.Point{}, draw.Src)
			}
		}
		chunks[chunk] = ebiten.NewImageFromImage(decoded)
	}
	return chunks
}

func decodeBobBank(data []byte, palette color.Palette) []*ebiten.Image {
	result := make([]*ebiten.Image, len(data)/bobBytes)
	for frame := range result {
		result[frame] = decodeBob(data[frame*bobBytes:(frame+1)*bobBytes], palette)
	}
	return result
}

func decodeBob(data []byte, palette color.Palette) *ebiten.Image {
	decoded := image.NewNRGBA(image.Rect(0, 0, bobSize, bobSize))
	for y := range bobSize {
		for x := range bobSize {
			mask := data[y*2+x/8] & (1 << uint(7-x%8))
			if mask == 0 {
				continue
			}
			value := uint8(0)
			for plane := range 4 {
				if data[(plane+1)*32+y*2+x/8]&(1<<uint(7-x%8)) != 0 {
					value |= 1 << plane
				}
			}
			decoded.Set(x, y, palette[value])
		}
	}
	return ebiten.NewImageFromImage(decoded)
}

func decodeActorBobBank(data []byte, palette color.Palette) ([]*ebiten.Image, error) {
	return decodeWideBobBank(data, actorBobHeight, actorBobBytes, "actor", palette)
}

func decodeOverlayBobBank(data []byte, palette color.Palette) ([]*ebiten.Image, error) {
	return decodeWideBobBank(data, overlayBobHeight, overlayBobBytes, "overlay", palette)
}

func decodeVehicleBobBank(data []byte, palette color.Palette) ([]*ebiten.Image, error) {
	return decodeWideBobBank(data, vehicleBobHeight, vehicleBobBytes, "vehicle", palette)
}

// decodeWideBobBank follows the original cookie-cut Blitter layout: one mask
// plane followed by four color planes, with each frame stored contiguously.
func decodeWideBobBank(data []byte, height, frameBytes int, kind string, palette color.Palette) ([]*ebiten.Image, error) {
	if len(data)%frameBytes != 0 {
		return nil, fmt.Errorf("%s bank size %d is not a multiple of %d", kind, len(data), frameBytes)
	}
	result := make([]*ebiten.Image, len(data)/frameBytes)
	for frame := range result {
		result[frame] = decodeWideBob(data[frame*frameBytes:(frame+1)*frameBytes], height, palette)
	}
	return result, nil
}

func decodeWideBob(data []byte, height int, palette color.Palette) *ebiten.Image {
	decoded := image.NewNRGBA(image.Rect(0, 0, actorBobWidth, height))
	planeBytes := actorBobWidth * height / 8
	for y := range height {
		for x := range actorBobWidth {
			bit := byte(1 << uint(7-x%8))
			byteOffset := y*actorBobWidth/8 + x/8
			if data[byteOffset]&bit == 0 {
				continue
			}
			value := uint8(0)
			for plane := range 4 {
				if data[(plane+1)*planeBytes+byteOffset]&bit != 0 {
					value |= 1 << plane
				}
			}
			decoded.Set(x, y, palette[value])
		}
	}
	return ebiten.NewImageFromImage(decoded)
}

func decodeFont(data []byte, palette color.Palette) *ebiten.Image {
	const glyphs = 46
	result := image.NewPaletted(image.Rect(0, 0, glyphs*8, 8), palette)
	for glyph := range glyphs {
		base := glyph * 64
		for y := range 8 {
			for x := range 8 {
				value := uint8(0)
				for plane := range 4 {
					if data[base+plane*16+y*2]&(1<<uint(7-x)) != 0 {
						value |= 1 << plane
					}
				}
				result.SetColorIndex(glyph*8+x, y, value)
			}
		}
	}
	return ebiten.NewImageFromImage(result)
}
