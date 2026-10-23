// Package runtime runs the supplied cartridge, not a guessed reconstruction of
// its rules. All CPU, VDP and audio synthesis code is implemented in Go.
package runtime

import (
	"battlecity/assets"
	core "battlecity/internal/mdcore"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
)

const Version = "0.1.5"
const (
	Up uint32 = 1 << iota
	Down
	Left
	Right
	A
	B
	C
	Start
)

type Machine struct {
	Core        core.Emulator
	Frame       uint64
	PAL         bool
	pixels      []byte
	menuState   []byte
	menuBattery []byte
}

func New(pal bool) (*Machine, error) {
	rom, err := assets.PlayableROM()
	if err != nil {
		return nil, err
	}
	m := &Machine{Core: core.NewEmulator(), PAL: pal}
	if pal {
		m.Core.SetRomPAL(rom)
	} else {
		m.Core.SetRom(rom)
	}
	m.Core.SetSixButton(false)
	m.Core.SetConsolePalette(true)
	m.Core.ConfigureBattleCityBattery()
	m.Core.SetP2Connected(true)
	m.Core.Start()
	return m, nil
}
func (m *Machine) FPS() int {
	if m.PAL {
		return 50
	}
	return 60
}
func (m *Machine) Step(p1, p2 uint32) []int16 {
	m.Core.SetInput(0, p1)
	m.Core.SetInput(1, p2)
	m.Core.RunFrame()
	m.Frame++
	if m.menuState == nil && m.AtMainMenu() {
		state, err := m.Save()
		if err == nil {
			m.menuState = state
			m.menuBattery = m.Core.GetSRAM()
		}
	}
	return m.Core.GetAudioSamples()
}

// AtMainMenu identifies the original menu event handler after scrolling ends.
func (m *Machine) AtMainMenu() bool {
	ram := m.Core.GetMainRAM()
	return binary.BigEndian.Uint32(ram[0x5e6:0x5ea]) == 0x693c && ram[0x1264] == 0xff
}

// SelectingStage uses the cartridge's actual state and tilemap, rather than a
// frame count or a pixel/color heuristic. It stops as soon as Enter is accepted.
func (m *Machine) SelectingStage() bool {
	ram := m.Core.GetMainRAM()
	if binary.BigEndian.Uint16(ram[0x600:0x602]) != 0xffff || binary.BigEndian.Uint32(ram[0x5e6:0x5ea]) != 0x33b0 {
		return false
	}
	for i := 0; i < 5; i++ {
		if m.ScreenTile(12+i, 13)&0x7ff != uint16(35+i) {
			return false
		}
	}
	return true
}

// ScreenTile follows the cartridge's plane width; H32 output does not imply a
// 32-tile VRAM stride (this game uses a wider tilemap).
func (m *Machine) ScreenTile(x, y int) uint16 {
	w := int(binary.BigEndian.Uint16(m.Core.GetMainRAM()[0x35e:0x360]))
	if w != 32 && w != 64 && w != 128 {
		return 0
	}
	return m.Core.PeekVRAMWord(uint16(0xc000 + (y*w+x)*2))
}

// ReturnToMenu restores a clean original menu without replaying the intro or
// terminating the app. Rebuild only if battery-backed high score has changed.
func (m *Machine) ReturnToMenu() error {
	if m.AtMainMenu() {
		return nil
	}
	battery := m.Core.GetSRAM()
	if len(m.menuState) > 0 && bytes.Equal(battery, m.menuBattery) {
		state := m.menuState
		if err := m.Load(state); err != nil {
			return err
		}
		m.Core.SetSRAM(battery)
		m.Step(0, 0)
		return nil
	}
	next, err := New(m.PAL)
	if err != nil {
		return err
	}
	next.Core.SetSRAM(battery)
	for i := 0; i < 900; i++ {
		next.Step(0, 0)
		if next.AtMainMenu() {
			next.Step(0, 0)
			*m = *next
			return nil
		}
	}
	return fmt.Errorf("main menu did not initialize")
}

// Image is a tightly packed RGBA image valid until the next Image call.
func (m *Machine) Image() *image.RGBA {
	w, h := m.Core.NativeWidth(), m.Core.GetActiveHeight()
	n := w * h * 4
	if len(m.pixels) != n {
		m.pixels = make([]byte, n)
	}
	src := m.Core.GetFramebuffer()
	stride := m.Core.GetFramebufferStride()
	for y := 0; y < h; y++ {
		copy(m.pixels[y*w*4:(y+1)*w*4], src[y*stride:y*stride+w*4])
	}
	return &image.RGBA{Pix: m.pixels, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
}
func (m *Machine) PixelHash() string {
	s := sha256.Sum256(m.Image().Pix)
	return hex.EncodeToString(s[:])
}
func (m *Machine) Save() ([]byte, error) {
	b, err := m.Core.Serialize()
	if err != nil {
		return nil, err
	}
	// The surrounding envelope identifies our fork, region and frontend counter.
	hdr := []byte(fmt.Sprintf("BCGO1 %t %d\n", m.PAL, m.Frame))
	return append(hdr, b...), nil
}
func (m *Machine) Load(b []byte) error {
	// Restore to a fresh machine first so a rejected state cannot partly modify
	// the running game. Region is part of the envelope, unlike upstream states.
	var pal bool
	var frame uint64
	var n int
	for n < len(b) && b[n] != '\n' && n < 80 {
		n++
	}
	if n == len(b) || n == 80 {
		return fmt.Errorf("invalid save header")
	}
	if _, err := fmt.Sscanf(string(b[:n]), "BCGO1 %t %d", &pal, &frame); err != nil {
		return fmt.Errorf("invalid save header: %w", err)
	}
	if pal != m.PAL {
		return fmt.Errorf("save uses a different region")
	}
	next, err := New(pal)
	if err != nil {
		return err
	}
	if err = next.Core.Deserialize(b[n+1:]); err != nil {
		return err
	}
	next.Frame = frame
	next.menuState = m.menuState
	next.menuBattery = m.menuBattery
	*m = *next
	return nil
}
