package main

import (
	"battlecity/assets"
	"battlecity/internal/audioqueue"
	machine "battlecity/internal/game"
	"battlecity/internal/storage"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type game struct {
	m           *machine.Game
	q           *audioqueue.Queue
	player      *audio.Player
	tex         *ebiten.Image
	dir         string
	muted, help bool
	aspect      string
	notice      string
	noticeUntil time.Time
	battery     []byte
	mixer       *audioqueue.Mixer
}

func (g *game) notify(s string) {
	g.notice = s
	g.noticeUntil = time.Now().Add(3 * time.Second)
	log.Print(s)
}
func pressed(k ebiten.Key) bool { return inpututil.IsKeyJustPressed(k) }
func key(k ebiten.Key) bool     { return ebiten.IsKeyPressed(k) }

func readPad(player int) uint32 {
	var b uint32
	keys := [2][8]ebiten.Key{
		{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyZ, ebiten.KeyX, ebiten.KeyC, ebiten.KeyEnter},
		{ebiten.KeyW, ebiten.KeyS, ebiten.KeyA, ebiten.KeyD, ebiten.KeyG, ebiten.KeyH, ebiten.KeyJ, ebiten.KeyTab},
	}
	for i, k := range keys[player] {
		if key(k) {
			b |= 1 << i
		}
	}
	// Space is an additional B button, not an automatic-fire modification.
	if player == 0 && key(ebiten.KeySpace) {
		b |= machine.B
	}
	ids := ebiten.AppendGamepadIDs(nil)
	if player < len(ids) {
		id := ids[player]
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			mapButtons := [8]ebiten.StandardGamepadButton{
				ebiten.StandardGamepadButtonLeftTop, ebiten.StandardGamepadButtonLeftBottom,
				ebiten.StandardGamepadButtonLeftLeft, ebiten.StandardGamepadButtonLeftRight,
				ebiten.StandardGamepadButtonRightLeft, ebiten.StandardGamepadButtonRightBottom,
				ebiten.StandardGamepadButtonRightRight, ebiten.StandardGamepadButtonCenterRight,
			}
			for i, k := range mapButtons {
				if ebiten.IsStandardGamepadButtonPressed(id, k) {
					b |= 1 << i
				}
			}
			x := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
			y := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
			if x < -0.35 {
				b |= machine.Left
			}
			if x > .35 {
				b |= machine.Right
			}
			if y < -0.35 {
				b |= machine.Up
			}
			if y > .35 {
				b |= machine.Down
			}
		}
	}
	// Resolve physically impossible opposite directions consistently.
	if b&(machine.Left|machine.Right) == machine.Left|machine.Right {
		b &^= machine.Left | machine.Right
	}
	if b&(machine.Up|machine.Down) == machine.Up|machine.Down {
		b &^= machine.Up | machine.Down
	}
	return b
}
func (g *game) savePath() string {
	if g.m.PAL {
		return filepath.Join(g.dir, "slot-native-pal.json")
	}
	return filepath.Join(g.dir, "slot-native-ntsc.json")
}
func (g *game) save() error {
	b, err := g.m.Save()
	if err != nil {
		return err
	}
	return storage.WriteAtomic(g.savePath(), b)
}
func (g *game) load() error {
	b, err := os.ReadFile(g.savePath())
	if err != nil {
		return err
	}
	if err = g.m.Load(b); err != nil {
		return err
	}
	g.q.Clear()
	g.mixer.Clear()
	return nil
}
func (g *game) Update() error {
	if pressed(ebiten.KeyF10) {
		return ebiten.Termination
	}
	returning := pressed(ebiten.KeyEscape)
	if returning {
		if g.m.AtMainMenu() {
			return ebiten.Termination
		}
		g.m.ReturnToMenu()
		g.mixer.Clear()
		g.q.Clear()
		g.help = false
		g.noticeUntil = time.Time{}
	}
	if pressed(ebiten.KeyF11) || (key(ebiten.KeyAltLeft) && pressed(ebiten.KeyEnter)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if pressed(ebiten.KeyF1) {
		g.help = !g.help
	}
	if pressed(ebiten.KeyM) {
		g.muted = !g.muted
		g.q.Mute(g.muted)
		g.notify(fmt.Sprintf("Sound muted: %t", g.muted))
	}
	if pressed(ebiten.KeyF5) {
		if err := g.save(); err != nil {
			g.notify("Save failed: " + err.Error())
		} else {
			g.notify("Saved (F9 to restore)")
		}
	}
	if pressed(ebiten.KeyF9) {
		if err := g.load(); err != nil {
			g.notify("Load failed: " + err.Error())
		} else {
			g.notify("Restored")
		}
	}
	if pressed(ebiten.KeyF8) {
		highscore := g.m.HighScore
		m, err := machine.New(g.m.PAL)
		if err != nil {
			return err
		}
		g.m = m
		g.m.HighScore = highscore
		g.mixer.Clear()
		g.q.Clear()
		g.notify("Game reset")
	}
	var p1, p2 uint32
	if !returning {
		p1, p2 = readPad(0), readPad(1)
	}
	g.m.Step(p1, p2)
	for _, event := range g.m.Events {
		g.mixer.Play(event.Name, event.Priority)
	}
	if !g.mixer.Busy() && g.m.Mode == machine.Playing && !g.m.Paused && !g.m.Lost {
		name := ""
		for i := 0; i < 2; i++ {
			if g.m.Tanks[i].HP != 0 {
				name = "engine_idle"
			}
		}
		for i := 0; i < 2; i++ {
			if g.m.Tanks[i].HP != 0 && g.m.Tanks[i].Speed != 0 {
				name = "engine_move"
			}
		}
		if name != "" {
			g.mixer.Play(name, 2)
		}
	}
	g.q.Push(g.mixer.Block(g.m.FPS()))
	if g.m.Frame%120 == 0 {
		g.persistBattery()
	}
	im := g.m.Image()
	if g.tex == nil || g.tex.Bounds().Dx() != im.Rect.Dx() || g.tex.Bounds().Dy() != im.Rect.Dy() {
		if g.tex != nil {
			g.tex.Deallocate()
		}
		g.tex = ebiten.NewImage(im.Rect.Dx(), im.Rect.Dy())
	}
	g.tex.WritePixels(im.Pix)
	return nil
}

func (g *game) persistBattery() {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, g.m.HighScore)
	binary.BigEndian.PutUint32(b[4:], ^g.m.HighScore)
	if len(g.battery) == 8 && string(b) == string(g.battery) {
		return
	}
	if err := storage.WriteAtomic(filepath.Join(g.dir, "highscore.sram"), b); err != nil {
		log.Print(err)
		return
	}
	g.battery = b
}
func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	if g.tex != nil {
		w, h := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
		tw, th := float64(g.tex.Bounds().Dx()), float64(g.tex.Bounds().Dy())
		sx, sy := 1.0, 1.0
		if g.aspect == "crt" {
			// 4:3 reproduces TV geometry; native pixels remain available with -aspect pixel.
			displayW := th * 4 / 3
			scale := min(w/displayW, h/th)
			sx = scale * displayW / tw
			sy = scale
		} else {
			scale := max(1, int(min(w/tw, h/th)))
			sx = float64(scale)
			sy = sx
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sx, sy)
		op.GeoM.Translate((w-tw*sx)/2, (h-th*sy)/2)
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(g.tex, op)
	}
	if g.help {
		ebitenutil.DrawRect(screen, 8, 8, 490, 190, color.RGBA{0, 0, 0, 230})
		ebitenutil.DebugPrintAt(screen, "BATTLE CITY - Native Go\n\nP1: Arrows / Z X C (or Space) / Enter\nP2: W A S D / G H J / Tab\nGamepads: D-pad / West South East / Start\nSTAGE: Enter select / Space next / C previous\nF1 Help  F5 Save  F9 Load  F8 Reset\nF11 Fullscreen  M Mute  F10 Quit\nEsc: Main menu / Quit from main menu\nIn-game pause: Enter (original Start button)", 16, 16)
	}
	if time.Now().Before(g.noticeUntil) {
		ebitenutil.DrawRect(screen, 0, float64(screen.Bounds().Dy()-22), float64(screen.Bounds().Dx()), 22, color.RGBA{0, 0, 0, 230})
		ebitenutil.DebugPrintAt(screen, g.notice, 6, screen.Bounds().Dy()-18)
	}
}

func (g *game) Layout(w, h int) (int, int) { return max(w, 320), max(h, 240) }

func run() error {
	pal := flag.Bool("pal", false, "50 updates/s; default is 60 updates/s")
	fullscreen := flag.Bool("fullscreen", false, "start fullscreen")
	scale := flag.Int("scale", 3, "initial integer window scale, 1..6")
	aspect := flag.String("aspect", "crt", "crt (4:3 television) or pixel (native square pixels)")
	mute := flag.Bool("mute", false, "start without sound")
	flag.Parse()
	if *aspect != "crt" && *aspect != "pixel" {
		return fmt.Errorf("-aspect must be crt or pixel")
	}
	if *scale < 1 || *scale > 6 {
		return fmt.Errorf("-scale must be 1..6")
	}
	dir, err := storage.Dir()
	if err != nil {
		return err
	}
	lf, err := os.OpenFile(filepath.Join(dir, "battlecity.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		defer lf.Close()
		log.SetOutput(lf)
	}
	m, err := machine.New(*pal)
	if err != nil {
		return err
	}
	if b, err := os.ReadFile(filepath.Join(dir, "highscore.sram")); err == nil && len(b) == 8 {
		score := binary.BigEndian.Uint32(b)
		if binary.BigEndian.Uint32(b[4:]) == ^score {
			m.HighScore = score
		}
	}
	m.RandomSeed ^= uint16(time.Now().UnixNano())
	q := audioqueue.New()
	q.Mute(*mute)
	ctx := audio.NewContext(48000)
	p, err := ctx.NewPlayer(q)
	if err != nil {
		return fmt.Errorf("audio: %w", err)
	}
	defer p.Close()
	p.SetBufferSize(32 * time.Millisecond)
	p.Play()
	g := &game{m: m, q: q, player: p, dir: dir, aspect: *aspect, muted: *mute}
	g.mixer, err = audioqueue.NewMixer(*pal)
	if err != nil {
		return err
	}
	defer g.persistBattery()
	resources, err := assets.Load()
	if err != nil {
		return fmt.Errorf("window icon: %w", err)
	}
	ebiten.SetWindowIcon([]image.Image{resources.Icon})
	ebiten.SetWindowTitle("Battle City")
	ebiten.SetWindowSize(256**scale, 224**scale)
	ebiten.SetWindowSizeLimits(320, 280, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(*fullscreen)
	// Ebitengine stops updates while unfocused. Drop queued sound at that point
	// instead of accumulating seconds of latency on returning to the game.
	ebiten.SetRunnableOnUnfocused(false)
	ebiten.SetTPS(m.FPS())
	return ebiten.RunGame(g)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		log.Print(err)
		os.Exit(1)
	}
}
