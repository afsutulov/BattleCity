package audioqueue

import (
	"battlecity/assets"
	"encoding/binary"
	"fmt"
)

type Mixer struct {
	clips              map[string][]int16
	current            []int16
	position, priority int
}

func NewMixer(pal bool) (*Mixer, error) {
	m := &Mixer{clips: make(map[string][]int16)}
	for _, name := range []string{"begin", "engine_idle", "engine_move", "player_explosion", "shot", "impact", "pause", "enemy_explosion", "brick", "bonus_appears", "bonus_collected", "game_over"} {
		b, e := assets.Sound(name, pal)
		if e != nil {
			return nil, e
		}
		if len(b) < 44 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
			return nil, fmt.Errorf("invalid WAV: %s", name)
		}
		var pcm []byte
		for pos := 12; pos+8 <= len(b); {
			size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
			end := pos + 8 + size
			if end > len(b) {
				return nil, fmt.Errorf("truncated WAV: %s", name)
			}
			if string(b[pos:pos+4]) == "fmt " && (size < 16 || binary.LittleEndian.Uint16(b[pos+8:]) != 1 || binary.LittleEndian.Uint16(b[pos+10:]) != 2 || binary.LittleEndian.Uint32(b[pos+12:]) != 48000 || binary.LittleEndian.Uint16(b[pos+22:]) != 16) {
				return nil, fmt.Errorf("unsupported WAV: %s", name)
			}
			if string(b[pos:pos+4]) == "data" {
				pcm = b[pos+8 : end]
			}
			pos = end + (size & 1)
		}
		if len(pcm) == 0 || len(pcm)%4 != 0 {
			return nil, fmt.Errorf("missing PCM data: %s", name)
		}
		clip := make([]int16, len(pcm)/2)
		for i := range clip {
			clip[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
		}
		m.clips[name] = clip
	}
	return m, nil
}
func (m *Mixer) Play(name string, priority int) {
	if m.Busy() && m.priority > priority {
		return
	}
	if clip, ok := m.clips[name]; ok {
		m.current = clip
		m.position = 0
		m.priority = priority
	}
}
func (m *Mixer) Busy() bool { return m.position < len(m.current) }
func (m *Mixer) Clear()     { m.current = nil; m.position = 0; m.priority = 0 }
func (m *Mixer) Block(fps int) []int16 {
	out := make([]int16, 48000/fps*2)
	n := copy(out, m.current[m.position:])
	m.position += n
	return out
}
