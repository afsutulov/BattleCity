package assets

import (
	"image"
	"strings"
)

// GameText renders the cartridge's own 8x8 uppercase glyphs, with an X added
// in the same style because the cartridge reuses its tile for a HUD graphic.
// Nonzero 4-bit pixels become black ink; zero pixels stay transparent.
func GameText(value string) *image.RGBA {
	value = strings.ToUpper(value)
	im := image.NewRGBA(image.Rect(0, 0, len(value)*8, 8))
	for n, c := range []byte(value) {
		if c == ' ' {
			continue
		}
		if c == 'X' {
			rows := [8]byte{0x63, 0x63, 0x36, 0x1c, 0x36, 0x63, 0x63, 0x00}
			for y, row := range rows {
				for x := 0; x < 8; x++ {
					if row&(0x80>>x) != 0 {
						im.Pix[y*im.Stride+(n*8+x)*4+3] = 255
					}
				}
			}
			continue
		}
		tile := original[0x6c68+int(c)*32 : 0x6c68+(int(c)+1)*32]
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				shift := uint(0)
				if x%2 == 0 {
					shift = 4
				}
				if (tile[y*4+x/2]>>shift)&15 != 0 {
					im.Pix[y*im.Stride+(n*8+x)*4+3] = 255
				}
			}
		}
	}
	return im
}
