package sound

import (
	"encoding/binary"
	"testing"

	gameassets "github.com/olivierh59500/ikariwarriors/assets"
)

func TestOriginalSongProducesContinuousPCM(t *testing.T) {
	data, err := gameassets.Original.ReadFile("original/audio-driver.bin")
	if err != nil {
		t.Fatal(err)
	}
	source := &stream{data: data, targetGain: 1, masterGain: 1}
	source.startSong(0)
	buffer := make([]byte, SampleRate*4*3)
	if n, err := source.Read(buffer); n != len(buffer) || err != nil {
		t.Fatalf("Read returned %d, %v", n, err)
	}
	nonzero, peak := 0, 0
	for offset := 0; offset < len(buffer); offset += 2 {
		value := int(int16(binary.LittleEndian.Uint16(buffer[offset:])))
		if value != 0 {
			nonzero++
		}
		if value < 0 {
			value = -value
		}
		peak = max(peak, value)
	}
	if nonzero < SampleRate || peak < 100 {
		t.Fatalf("translated song produced %d nonzero samples with peak %d", nonzero, peak)
	}
	small := make([]byte, 4096)
	if allocations := testing.AllocsPerRun(50, func() { _, _ = source.Read(small) }); allocations != 0 {
		t.Fatalf("PCM renderer allocates %.1f times per read", allocations)
	}
}

func TestSongTableInitializesAllFourVoices(t *testing.T) {
	data, err := gameassets.Original.ReadFile("original/audio-driver.bin")
	if err != nil {
		t.Fatal(err)
	}
	source := &stream{data: data}
	source.startSong(0)
	wantOrders := [...]int{0x640, 0x664, 0x670, 0x684}
	for index, value := range source.voices {
		if value.order != wantOrders[index] || value.pattern <= 0 || value.duration != 1 {
			t.Fatalf("voice %d order=%x pattern=%x duration=%d", index, value.order, value.pattern, value.duration)
		}
	}
}

func TestAllOriginalSongEntriesProduceAudio(t *testing.T) {
	data, err := gameassets.Original.ReadFile("original/audio-driver.bin")
	if err != nil {
		t.Fatal(err)
	}
	for track := 0; track < maximumSongCount; track++ {
		source := &stream{data: data, targetGain: 1, masterGain: 1}
		source.startSong(track)
		buffer := make([]byte, SampleRate*4)
		_, _ = source.Read(buffer)
		nonzero := false
		for _, value := range buffer {
			if value != 0 {
				nonzero = true
				break
			}
		}
		if !nonzero {
			t.Fatalf("original song entry %d remained silent", track)
		}
	}
}
