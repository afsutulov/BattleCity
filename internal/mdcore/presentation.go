package core

// NativeWidth reports pixels before display aspect-ratio correction. The fork
// keeps H32 output at 256 pixels, rather than stretching it inside the core.
func (e *Emulator) NativeWidth() int { return e.vdp.ActiveWidth() }

// PeekVRAMWord observes a tilemap without changing VDP ports or CPU execution.
func (e *Emulator) PeekVRAMWord(address uint16) uint16 {
	return uint16(e.vdp.vram[address])<<8 | uint16(e.vdp.vram[address+1])
}

func (e *Emulator) SetConsolePalette(enabled bool) { e.vdp.consolePalette = enabled }

// ConfigureBattleCityBattery supplies the eight bytes accessed directly by this
// cartridge at $200000. Its "RA" header has space-filled bounds, so generic
// header-based SRAM detection cannot find them. The ROM itself stays unchanged.
func (e *Emulator) ConfigureBattleCityBattery() {
	e.bus.sram = make([]byte, 8)
	for i := range e.bus.sram {
		e.bus.sram[i] = 0xff
	}
	e.bus.sramStart = 0x200000
	e.bus.sramEnd = 0x200007
	e.bus.sramEnabled = true
	e.bus.sramWritable = true
}

// SetRomPAL selects European hardware without changing a byte of the cartridge.
func (e *Emulator) SetRomPAL(rom []byte) { e.setRom(rom, VideoPAL, ConsoleEurope) }
