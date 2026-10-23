package game

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
)

var black = color.RGBA{0, 0, 0, 255}
var white = color.RGBA{238, 238, 238, 255}
var grey = color.RGBA{136, 136, 136, 255}
var orange = color.RGBA{204, 136, 34, 255}

func (g *Game) tileAt(t, pal, x, y int) {
	if t < 0 || t >= 784 || pal < 0 || pal >= len(g.resources.Tiles) {
		return
	}
	if t == 18 && pal == 0 && g.WaterPhase > 0 {
		draw.Draw(g.pixels, image.Rect(x, y, x+8, y+8), g.resources.Water[(g.WaterPhase-1)%2], image.Point{}, draw.Over)
		return
	}
	src := g.resources.Tiles[pal]
	p := image.Pt(t%32*8, t/32*8)
	draw.Draw(g.pixels, image.Rect(x, y, x+8, y+8), src, p, draw.Over)
}
func (g *Game) cell(t, pal, x, y int) {
	draw.Draw(g.pixels, image.Rect(x, y, x+8, y+8), image.NewUniform(black), image.Point{}, draw.Src)
	g.tileAt(t, pal, x, y)
}
func (g *Game) sprite(t, w, h, pal, x, y int) {
	// The exported frames retain the original column order; drawing them is
	// ordinary PNG compositing, with no video registers or emulated hardware.
	for col := 0; col < w; col++ {
		for row := 0; row < h; row++ {
			g.tileAt(t+col*h+row, pal, x+col*8, y+row*8)
		}
	}
}
func (g *Game) text(s string, x, y int, ink color.RGBA) {
	for i, c := range []byte(strings.ToUpper(s)) {
		if c >= 128 || c == ' ' {
			continue
		}
		draw.DrawMask(g.pixels, image.Rect(x+i*8, y, x+i*8+8, y+8), image.NewUniform(ink), image.Point{}, g.resources.Font, image.Pt(int(c)%16*8, int(c)/16*8), draw.Over)
	}
}
func (g *Game) number(value uint32, x, y int, ink color.RGBA) {
	s := strconv.FormatUint(uint64(value), 10)
	g.text(s, x-(len(s)-1)*8, y, ink)
}
func (g *Game) ui(m [][]uint16, pal int, offsetY int) {
	for y, row := range m {
		for x, t := range row {
			g.cell(int(t)&2047, pal, 16+x*8, 8+y*8+offsetY)
		}
	}
}
func (g *Game) drawMenu() {
	offset := max(0, 224-g.Age)
	if g.Mode == Closing {
		offset = 0
	}
	g.ui(g.resources.Maps.Menu, 4, offset)
	g.number(g.HighScore, 152, 16+offset, white)
	g.number(g.Tanks[0].Score, 72, 16+offset, white)
	if offset == 0 {
		g.sprite(256+3*8+int((g.Frame>>2)&1)*4, 2, 2, 1, 65, 125+g.Selection*16)
	}
}
func (g *Game) drawStage() {
	draw.Draw(g.pixels, g.pixels.Bounds(), image.NewUniform(grey), image.Point{}, draw.Src)
	for i := 0; i < 5; i++ {
		g.cell(35+i, 0, 96+i*8, 104)
	}
	n := g.StageNumber + 1
	if n >= 10 {
		g.cell(110+n/10%10, 0, 144, 104)
	}
	g.cell(110+n%10, 0, 152, 104)
	if g.Mode == Stage {
		for i, s := range []string{"ENTER  SELECT", "SPACE  NEXT STAGE", "C  PREVIOUS STAGE"} {
			g.text(s, (256-len(s)*8)/2, 152+i*16, black)
		}
	}
}
func (g *Game) drawTank(t Tank) {
	if t.HP == 0 {
		return
	}
	if t.Freeze != 0 && (g.Frame-1)&31 < 16 {
		return
	}
	x, y := 16+t.X, 8+t.Y
	if t.Birth != 0 {
		g.sprite(160+(t.Birth&12), 2, 2, 0, x, y)
		return
	}
	pal := t.Color
	if t.Bonus != 0 {
		pal = 3
		if (g.Frame-1)&15 < 8 {
			pal = 0
		}
	} else {
		if t.HP > 1 && (g.Frame-1)&1 != 0 {
			pal = 1
			if t.HP == 4 {
				pal = 2
			}
		}
		if t.HP == 2 && (g.Frame-1)&1 == 0 {
			pal = 2
		}
	}
	frame := 0
	if t.Speed != 0 && !g.Paused {
		frame = int(g.Frame-1) & 4
	}
	g.sprite(256+t.Type*32+t.Direction*8+frame, 2, 2, pal, x, y)
	if t.Shield != 0 {
		g.sprite(200+((t.Shield*2)&4), 2, 2, 0, x, y)
	}
}
func (g *Game) drawWorld() {
	draw.Draw(g.pixels, g.pixels.Bounds(), image.NewUniform(grey), image.Point{}, draw.Src)
	draw.Draw(g.pixels, image.Rect(16, 8, 224, 216), image.NewUniform(black), image.Point{}, draw.Src)
	for y, row := range g.Map {
		for x, t := range row {
			if t&0x8000 == 0 {
				g.cell(int(t)&2047, 0, 16+x*8, 8+y*8)
			}
		}
	}
	if g.Mode == Opening {
		h := max(0, 14-g.Age) * 8
		draw.Draw(g.pixels, image.Rect(0, 0, 256, h), image.NewUniform(grey), image.Point{}, draw.Src)
		draw.Draw(g.pixels, image.Rect(0, 224-h, 256, 224), image.NewUniform(grey), image.Point{}, draw.Src)
		return
	}
	if g.Bonus.Kind != 0 && (g.Frame-1)&15 > 7 {
		g.sprite(124+g.Bonus.Kind*4, 2, 2, 0, 16+g.Bonus.X, 8+g.Bonus.Y)
	}
	for i := 7; i >= 0; i-- {
		g.drawTank(g.Tanks[i])
	}
	explosions := g.Explosions
	if g.VisualReady {
		explosions = g.VisualExplosions
	}
	for _, e := range explosions {
		if e.Kind == 0 {
			continue
		}
		if e.Frame <= 2 {
			g.sprite(240+e.Frame*4, 2, 2, 0, 16+e.X, 8+e.Y)
		} else {
			g.sprite(160+e.Frame*16, 4, 4, 0, 8+e.X, e.Y)
		}
	}
	for i := 15; i >= 0; i-- {
		b := g.Bullets[i]
		if b.Speed != 0 {
			g.sprite(176+b.Direction*2, 1, 2, 0, 19+b.X, 8+b.Y)
		}
	}
	floating := g.Float
	if g.VisualReady {
		floating = g.VisualFloat
	}
	if floating.Timer != 0 {
		tile := 184 + floating.Value*4
		if floating.Value == 4 {
			tile = 58
		}
		g.sprite(tile, 2, 2, 0, 16+floating.X, 8+floating.Y)
	}
	for y, row := range g.Map {
		for x, t := range row {
			if t&0x8000 != 0 {
				g.tileAt(int(t)&2047, 0, 16+x*8, 8+y*8)
			}
		}
	}
	for i := 0; i < g.Remaining; i++ {
		g.cell(106, 0, 232+(i%2)*8, 16+(i/2)*8)
	}
	for p := 0; p < 2; p++ {
		if p == 1 && !g.TwoPlayers {
			continue
		}
		y := 120 + p*24
		g.cell(88, 0, 232, y)
		g.cell(19, 0, 240, y)
		g.cell(20, 0, 232, y+8)
		g.cell(110+min(9, max(0, g.Tanks[p].Lives)), 0, 240, y+8)
	}
	g.cell(108, 0, 232, 176)
	g.cell(109, 0, 232, 184)
	g.cell(44, 0, 240, 176)
	g.cell(45, 0, 240, 184)
	n := g.StageNumber + 1
	if n >= 10 {
		g.cell(110+n/10%10, 0, 232, 192)
	}
	g.cell(110+n%10, 0, 240, 192)
	if g.Paused && g.WaterFrame&32 != 0 {
		g.sprite(22, 4, 1, 0, 100, 112)
		g.sprite(23, 4, 1, 0, 108, 112)
	}

	if g.Lost {
		y := max(104, 208-g.Age) + 8
		g.sprite(120, 4, 1, 0, 104, y)
		g.sprite(124, 4, 1, 0, 104, y+8)
	}

}
func (g *Game) drawScore() {
	m := g.resources.Maps.Score1
	if g.TwoPlayers {
		m = g.resources.Maps.Score2
	}
	for y, row := range m {
		for x, t := range row {
			pal := 0
			if y == 1 {
				if x <= 12 {
					pal = 6
				} else {
					pal = 5
				}
			}
			if y == 5 {
				pal = 6
			}
			if y == 7 {
				pal = 5
			}
			g.cell(int(t)&2047, pal, 16+x*8, 8+y*8)
		}
	}

	g.number(g.HighScore, 176, 16, orange)
	g.number(uint32(g.StageNumber+1), 144, 32, white)
	for p := 0; p < 2; p++ {
		if p == 1 && !g.TwoPlayers {
			continue
		}
		shift := p * 144
		g.number(g.Tanks[p].Score, 72+shift, 64, orange)
		total := 0
		for i, n := range g.Kills[p] {
			shown := n
			total += shown
			g.number(uint32(shown), 96+p*48, 88+i*24, white)
			g.number(uint32(shown*(i+1)*100), 40+shift, 88+i*24, white)
		}
		g.number(uint32(total), 96+p*48, 176, white)
	}
	for i := 0; i < 4; i++ {
		g.sprite(256+(i+4)*32, 2, 2, 3, 113, 85+i*24)
	}
}
func (g *Game) drawEditor() {
	draw.Draw(g.pixels, g.pixels.Bounds(), image.NewUniform(grey), image.Point{}, draw.Src)
	draw.Draw(g.pixels, image.Rect(16, 8, 224, 216), image.NewUniform(black), image.Point{}, draw.Src)
	for y, row := range g.CustomMap {
		for x, t := range row {
			g.cell(int(t)&2047, 0, 16+x*8, 8+y*8)
		}
	}
	if (g.Frame-1)&16 == 0 {
		g.sprite(256, 2, 2, 1, 16+g.EditorX*16, 8+g.EditorY*16)
	}
}

// Image returns the current native 256x224 frame; simulation never runs here.
func (g *Game) Image() *image.RGBA {
	if g.pixels == nil {
		g.pixels = image.NewRGBA(image.Rect(0, 0, Width, Height))
	}
	draw.Draw(g.pixels, g.pixels.Bounds(), image.NewUniform(black), image.Point{}, draw.Src)
	switch g.Mode {
	case Closing:
		if g.ClosingFrom == Score {
			g.drawScore()
		} else {
			g.drawMenu()
		}
		h := min(112, g.Age*8)
		draw.Draw(g.pixels, image.Rect(0, 0, 256, h), image.NewUniform(grey), image.Point{}, draw.Src)
		draw.Draw(g.pixels, image.Rect(0, 224-h, 256, 224), image.NewUniform(grey), image.Point{}, draw.Src)
	case Menu:
		g.drawMenu()
	case Stage, Intro:
		g.drawStage()
	case Opening, Playing:
		g.drawWorld()
	case Score:
		g.drawScore()
	case GameOver:
		g.ui(g.resources.Maps.GameOver, 0, 0)
	case Editor:
		g.drawEditor()
	default:
		g.text(fmt.Sprintf("STATE %d", g.Mode), 8, 8, white)
	}
	return g.pixels
}
