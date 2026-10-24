// Package sound translates Ikari Warriors' native four-channel Paula driver.
package sound

import (
	"encoding/binary"
	"io"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	gameassets "github.com/olivierh59500/ikariwarriors/assets"
)

const (
	SampleRate       = 44100
	paulaClock       = 3546895.0
	driverTickRate   = 50.0
	songTableOffset  = 0x6cf
	noteTableOffset  = 0x2c8
	envelopeMap      = 0x70d
	envelopeData     = 0x719
	instrumentTable  = 0x1086
	effectTable      = 0xcbe
	instrumentCount  = 13
	maximumSongCount = 3
)

var (
	contextOnce sync.Once
	context     *audio.Context
)

// Audio owns the translated sequencer and its continuous PCM player.
type Audio struct {
	stream  *stream
	player  *audio.Player
	current string
	muted   bool
	paused  bool
}

type voice struct {
	flags          uint8
	orderIndex     uint8
	pattern        int
	order          int
	orderEnd       uint8
	orderLoop      uint8
	pitchDelta     int16
	pitchSlide     int16
	envelope       uint8
	envelopeTicks  uint8
	volume         uint8
	volumeScale    uint8
	duration       uint8
	durationSet    uint8
	slideDelay     uint8
	note           uint8
	vibratoDepth   uint8
	vibratoStep    uint8
	vibratoPhase   uint8
	transpose      int8
	instrument     uint8
	envelopeShape  uint8
	envelopeCursor uint8
	period         int
	sample         sampleState
}

type sampleState struct {
	start     int
	end       int
	loopStart int
	loop      bool
	position  float64
	active    bool
}

type stream struct {
	mu         sync.Mutex
	data       []byte
	voices     [4]voice
	active     bool
	tempo      uint8
	tempoPhase uint8
	tickPhase  float64
	masterGain float64
	targetGain float64
}

// New starts a click-free stream from the original custom driver block.
func New(muted bool) (*Audio, error) {
	data, err := gameassets.Original.ReadFile("original/audio-driver.bin")
	if err != nil {
		return nil, err
	}
	contextOnce.Do(func() { context = audio.NewContext(SampleRate) })
	source := &stream{data: append([]byte(nil), data...)}
	player, err := context.NewPlayer(source)
	if err != nil {
		return nil, err
	}
	player.SetVolume(0.68)
	if !muted {
		player.Play()
	}
	return &Audio{stream: source, player: player, muted: muted}, nil
}

// Music starts one of the original song entries used by the game states.
func (a *Audio) Music(name string) {
	if a == nil || a.current == name {
		return
	}
	a.current = name
	track := 0
	if name == "victory" {
		track = 1
	} else if name == "ending" {
		track = 2
	}
	a.stream.mu.Lock()
	a.stream.startSong(track)
	if a.muted || a.paused {
		a.stream.targetGain = 0
	} else {
		a.stream.targetGain = 1
	}
	a.stream.mu.Unlock()
	if !a.muted && !a.paused {
		a.player.Play()
	}
}

// Effect injects one original effect sequence on the requested Paula channel.
func (a *Audio) Effect(index, channel int) {
	if a == nil {
		return
	}
	a.stream.mu.Lock()
	a.stream.startEffect(index, channel)
	a.stream.mu.Unlock()
}

// SetPaused follows gameplay pause without losing sequence position.
func (a *Audio) SetPaused(paused bool) {
	if a == nil || a.paused == paused {
		return
	}
	a.paused = paused
	a.stream.mu.Lock()
	if paused || a.muted {
		a.stream.targetGain = 0
	} else {
		a.stream.targetGain = 1
	}
	a.stream.mu.Unlock()
	if paused || a.muted {
		a.player.Pause()
	} else {
		a.player.Play()
	}
}

// Close releases the Ebitengine audio player.
func (a *Audio) Close() {
	if a != nil && a.player != nil {
		_ = a.player.Close()
	}
}

func (s *stream) startSong(track int) {
	track = max(0, min(maximumSongCount-1, track))
	for index := range s.voices {
		s.voices[index] = voice{volumeScale: 255, duration: 1}
	}
	record := songTableOffset + track*18
	if record+17 >= len(s.data) {
		s.active = false
		return
	}
	s.tempo = s.data[record]
	s.tempoPhase = 255
	for index := range s.voices {
		entry := record + 1 + index*4
		orderWord := binary.BigEndian.Uint16(s.data[entry:])
		order := int(binary.BigEndian.Uint16(s.data[entry+2:]))
		voice := &s.voices[index]
		voice.orderEnd = uint8(orderWord >> 8)
		voice.orderLoop = uint8(orderWord)
		voice.order = order
		if order+1 < len(s.data) {
			voice.pattern = int(binary.BigEndian.Uint16(s.data[order:]))
		}
	}
	s.active = true
	s.targetGain = 1
}

func (s *stream) startEffect(index, channel int) {
	if channel < 0 || channel >= len(s.voices) || index < 0 {
		return
	}
	offset := effectTable + index*18
	if offset+18 > len(s.data) {
		return
	}
	raw := s.data[offset : offset+18]
	voice := &s.voices[channel]
	voice.flags = raw[0]
	voice.orderIndex = raw[1]
	voice.pattern = int(binary.BigEndian.Uint16(raw[2:]))
	voice.order = int(binary.BigEndian.Uint16(raw[4:]))
	orderWord := binary.BigEndian.Uint16(raw[6:])
	voice.orderEnd, voice.orderLoop = uint8(orderWord>>8), uint8(orderWord)
	voice.pitchDelta = int16(binary.BigEndian.Uint16(raw[8:]))
	voice.pitchSlide = int16(binary.BigEndian.Uint16(raw[10:]))
	voice.envelopeShape = raw[13]
	voice.envelopeTicks = raw[14]
	voice.volume = raw[15]
	voice.volumeScale = raw[16]
	voice.duration = raw[17]
	voice.transpose = -1
}

func (s *stream) Read(buffer []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	frames := len(buffer) / 4
	for frame := range frames {
		s.tickPhase += driverTickRate / SampleRate
		for s.tickPhase >= 1 {
			s.tickPhase--
			s.tick()
		}
		s.masterGain += (s.targetGain - s.masterGain) * 0.0018
		left, right := 0.0, 0.0
		for channel := range s.voices {
			value := s.voices[channel].sampleValue(s.data) * float64(s.voices[channel].finalVolume()) / 64
			if channel == 0 || channel == 3 {
				left += value
			} else {
				right += value
			}
		}
		writePCM(buffer[frame*4:], left*s.masterGain*0.42, right*s.masterGain*0.42)
	}
	clear(buffer[frames*4:])
	return len(buffer), nil
}

func (s *stream) tick() {
	if !s.active {
		return
	}
	for index := range s.voices {
		s.updateVoice(&s.voices[index])
	}
	previous := s.tempoPhase
	s.tempoPhase += s.tempo
	if s.tempoPhase >= previous {
		return
	}
	for index := range s.voices {
		s.stepSequence(&s.voices[index])
	}
}

func (s *stream) updateVoice(value *voice) {
	s.stepEnvelope(value)
	note := int(value.note) + int(value.transpose)
	note = max(0, min(127, note))
	instrument := int(value.instrument)
	if instrument < 0 || instrument >= instrumentCount {
		instrument = 0
	}
	record := instrumentTable + instrument*10
	tuning := uint32(binary.BigEndian.Uint16(s.data[record:]))
	periodValue := uint32(binary.BigEndian.Uint16(s.data[noteTableOffset+note*2:]))
	period := int(periodValue * tuning * 8 >> 16)
	if value.flags&0x04 != 0 {
		if value.slideDelay == 0 {
			value.pitchSlide += value.pitchDelta
			period += int(value.pitchSlide)
		} else {
			value.slideDelay--
		}
	}
	if value.flags&0x40 != 0 && value.vibratoDepth != 0 {
		value.vibratoPhase += max(1, value.vibratoStep)
		triangle := int(value.vibratoPhase&31) - 16
		if triangle < 0 {
			triangle = -triangle
		}
		period += (triangle - 8) * int(value.vibratoDepth) / 8
	}
	value.period = max(1, period)
}

func (s *stream) stepSequence(value *voice) {
	value.duration--
	if value.duration != 0 {
		if value.flags&0x08 != 0 {
			if value.flags&0x80 != 0 {
				value.note--
			} else {
				value.note++
			}
		}
		return
	}
	value.flags &= 0x50
	pointer := value.pattern
	for pointer >= 0 && pointer < len(s.data) {
		command := s.data[pointer]
		pointer++
		switch {
		case command >= 0xe0:
			value.durationSet = command + 0x21
			continue
		case command >= 0xc0:
			value.envelopeShape = command + 0x40
			continue
		case command >= 0xa0:
			value.instrument = command + 0x60
			continue
		case command >= 0x90:
			index := int(command - 0x90)
			if envelopeMap+index < len(s.data) {
				value.envelope = s.data[envelopeMap+index]
			}
			continue
		case command >= 0x80:
			var done bool
			pointer, done = s.sequenceControl(value, command, pointer)
			if done {
				value.pattern = pointer
				value.duration = value.durationSet
				return
			}
			continue
		default:
			value.note = command
			if value.flags&0x02 == 0 {
				s.startEnvelope(value)
			}
			value.envelopeCursor = value.envelope
			if value.flags&0x20 == 0 {
				s.startSample(value)
			}
			value.duration = value.durationSet
			value.pattern = pointer
			return
		}
	}
	value.sample.active = false
}

func (s *stream) sequenceControl(value *voice, command uint8, pointer int) (int, bool) {
	read := func() uint8 {
		if pointer >= len(s.data) {
			return 0
		}
		result := s.data[pointer]
		pointer++
		return result
	}
	switch command {
	case 0x80:
		value.instrument = 0
		value.sample.active = false
		return pointer, true
	case 0x81:
		value.flags = 0
	case 0x82:
		value.pitchDelta = int16(uint16(read())<<8 | uint16(read()))
		value.pitchSlide = 0
		value.flags |= 0x04
		value.slideDelay = read()
	case 0x83:
		value.flags |= 0x80
	case 0x84:
		value.flags |= 0x08
	case 0x85:
		value.orderIndex += 2
		if value.orderIndex == value.orderEnd {
			value.orderIndex = value.orderLoop
		}
		order := value.order + int(value.orderIndex)
		if order+1 < len(s.data) {
			pointer = int(binary.BigEndian.Uint16(s.data[order:]))
		}
	case 0x86:
		value.vibratoStep = read()
		value.vibratoPhase = read()
		value.vibratoDepth = value.vibratoPhase * 2
		value.flags |= 0x40
	case 0x87:
		// The original target is an RTS and intentionally performs no update.
	case 0x88:
		value.transpose = int8(read())
	case 0x89:
		_ = read()
	case 0x8a:
		value.flags |= 0x20
	case 0x8b:
		_ = read()
		_ = read()
		_ = read()
		value.instrument = 1
	case 0x8c:
		value.volumeScale = read()
	case 0x8d:
		value.flags |= 0x02
	}
	return pointer, false
}

func (s *stream) startEnvelope(value *voice) {
	index := int(value.envelopeShape)
	if envelopeMap+index >= len(s.data) {
		value.volume = 0
		return
	}
	offset := int(s.data[envelopeMap+index])
	value.envelope = uint8(offset)
	value.envelopeTicks = 1
	initial := envelopeMap + 14 + offset
	if initial < len(s.data) {
		value.volume = s.data[initial]
	}
}

func (s *stream) stepEnvelope(value *voice) {
	if value.envelopeTicks == 0 {
		return
	}
	value.envelopeTicks--
	if value.envelopeTicks != 0 {
		return
	}
	offset := envelopeData + int(value.envelope)
	if offset+1 >= len(s.data) {
		return
	}
	value.envelopeTicks = s.data[offset+1]
	if value.envelopeTicks == 0 {
		return
	}
	value.envelope += 2
	value.volume += s.data[offset]
}

func (s *stream) startSample(value *voice) {
	instrument := int(value.instrument)
	if instrument < 0 || instrument >= instrumentCount {
		instrument = 0
	}
	record := instrumentTable + instrument*10
	start := int(binary.BigEndian.Uint32(s.data[record+2:]))
	end := int(binary.BigEndian.Uint16(s.data[record+6:]))
	loop := int(int16(binary.BigEndian.Uint16(s.data[record+8:])))
	if start < 0 || start >= len(s.data) || end <= 0 || start+end > len(s.data) {
		value.sample.active = false
		return
	}
	value.sample = sampleState{start: start, end: end, loopStart: max(0, loop), loop: loop >= 0, active: true}
}

func (v *voice) finalVolume() int {
	return int(uint32(v.volume>>2) * 255 * uint32(v.volumeScale) >> 16)
}

func (v *voice) sampleValue(data []byte) float64 {
	if !v.sample.active || v.period <= 0 {
		return 0
	}
	index := v.sample.start + int(v.sample.position)
	if index < 0 || index >= len(data) {
		v.sample.active = false
		return 0
	}
	value := float64(int8(data[index])) * 256
	v.sample.position += paulaClock / float64(v.period) / SampleRate
	for v.sample.position >= float64(v.sample.end) {
		if !v.sample.loop {
			v.sample.active = false
			break
		}
		v.sample.position = float64(v.sample.loopStart) + math.Mod(v.sample.position-float64(v.sample.loopStart), float64(v.sample.end-v.sample.loopStart))
	}
	return value
}

func writePCM(target []byte, left, right float64) {
	left = max(-32768, min(32767, left))
	right = max(-32768, min(32767, right))
	binary.LittleEndian.PutUint16(target, uint16(int16(left)))
	binary.LittleEndian.PutUint16(target[2:], uint16(int16(right)))
}

var _ io.Reader = (*stream)(nil)
