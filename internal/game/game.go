// Package game implements Battle City's rules using tanks, bullets and map
// cells. All simulation is native Go, advanced once per fixed game tick.
package game

import (
	"battlecity/assets"
	"encoding/json"
	"fmt"
	"image"
)

const Version = "0.2.0"
const Width, Height = 256, 224
const MapSize = 26
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

type Mode uint8

const (
	Menu Mode = iota
	Stage
	Intro
	Opening
	Playing
	Score
	GameOver
	Editor
	Closing
)

type Tank struct {
	X, Y, Color, Direction, HP, Shield, Speed, Lives, Birth           int
	Fire, Bullets, Freeze, BulletSpeed, BulletLimit, Bonus, Collision int
	Type, FireTimer, SteelShot, Ice                                   int
	Score                                                             uint32
}
type Bullet struct{ X, Y, Direction, Speed, Owner int }
type Explosion struct{ X, Y, Frame, Strobe, Kind int }
type Bonus struct{ X, Y, Kind int }
type FloatingScore struct{ X, Y, Timer, Value int }
type Sound struct {
	Name     string
	Priority int
}
type Game struct {
	Mode                                                               Mode
	Frame                                                              uint64
	WaterFrame                                                         uint16
	WaterPhase                                                         int
	VisualExplosions                                                   [16]Explosion
	VisualFloat                                                        FloatingScore
	VisualReady                                                        bool
	PAL                                                                bool
	ClosingFrom, ClosingTo                                             Mode
	Age                                                                int
	Selection, StageNumber                                             int
	TwoPlayers                                                         bool
	HighScore                                                          uint32
	Map                                                                [26][26]uint16
	Tanks                                                              [8]Tank
	Bullets                                                            [16]Bullet
	Explosions                                                         [16]Explosion
	Bonus                                                              Bonus
	Float                                                              FloatingScore
	Kills                                                              [2][4]int
	SpeedCounter                                                       uint8
	SpawnTimer, SpawnDelay, Remaining, Freeze, BaseArmor, VictoryTimer int
	Lost, Won, Paused                                                  bool
	LevelTicks                                                         int
	OldPad                                                             [2]uint32
	FireHeld                                                           [2]bool
	RandomSeed                                                         uint16
	RandomClock                                                        uint32
	Edited                                                             bool
	CustomMap                                                          [26][26]uint16
	EditorX, EditorY, EditorBrush, Repeat                              int
	EditorMoved                                                        bool
	Events                                                             []Sound `json:"-"`
	resources                                                          *assets.Resources
	pixels                                                             *image.RGBA
}

var directionX = [4]int{0, -1, 0, 1}
var directionY = [4]int{-1, 0, 1, 0}

func New(pal bool) (*Game, error) {
	r, err := assets.Load()
	if err != nil {
		return nil, err
	}
	g := &Game{PAL: pal, HighScore: 20000, RandomSeed: 0xd94b, resources: r}
	g.ReturnToMenu()
	g.Age = 0
	return g, nil
}
func (g *Game) FPS() int {
	if g.PAL {
		return 50
	}
	return 60
}
func (g *Game) AtMainMenu() bool     { return g.Mode == Menu }
func (g *Game) SelectingStage() bool { return g.Mode == Stage }
func (g *Game) ReturnToMenu() {
	g.Mode = Menu
	g.Age = 224
	g.Selection = 0
	g.Paused = false
	g.OldPad = [2]uint32{}
	g.Events = nil
	g.Lost = false
	g.Won = false
}
func (g *Game) play(name string, priority int) { g.Events = append(g.Events, Sound{name, priority}) }
func (g *Game) random() uint16 {
	// The original generator folds a changing video counter into a 16-bit
	// xorshift. A logical counter replaces that hardware entropy source.
	// The desktop entry point supplies a varying initial seed.
	g.RandomClock += 137
	s := (g.RandomSeed >> 1) ^ g.RandomSeed ^ uint16(g.RandomClock)
	g.RandomSeed = s ^ (s << 1)
	return g.RandomSeed
}
func (g *Game) Step(p1, p2 uint32) {
	g.Events = g.Events[:0]
	g.Frame++
	g.Age++
	pad := [2]uint32{p1, p2}
	click := [2]uint32{p1 &^ g.OldPad[0], p2 &^ g.OldPad[1]}
	switch g.Mode {
	case Menu:
		if g.Age < 224 {
			break
		}
		if click[0]&Down != 0 {
			g.Selection = (g.Selection + 1) % 3
		}
		if click[0]&Up != 0 {
			g.Selection = (g.Selection + 2) % 3
		}
		if click[0]&Start != 0 {
			g.Age = 0
			if g.Selection == 2 {
				g.openEditor()
			} else {
				g.TwoPlayers = g.Selection == 1
				g.ClosingFrom = Menu
				g.ClosingTo = Stage
				g.Mode = Closing
				g.StageNumber = 0
			}
		}
	case Closing:
		if g.Age >= 15 {
			g.Mode = g.ClosingTo
			g.Age = 0
			if g.Mode == Intro {
				g.play("begin", 10)
			}
		}
	case Stage:
		if g.Repeat > 0 {
			g.Repeat--
		}
		if g.Repeat == 0 {
			if p1&B != 0 {
				g.StageNumber = min(34, g.StageNumber+1)
				g.Repeat = 10
			}
			if p1&C != 0 {
				g.StageNumber = max(0, g.StageNumber-1)
				g.Repeat = 10
			}
		}
		if click[0]&Start != 0 {
			g.initGame()
			g.Mode = Intro
			g.Age = 0
			g.play("begin", 10)
		}
	case Intro:
		if g.Age >= 64 {
			g.initLevel()
			g.Mode = Opening
			g.Age = 0
		}
	case Opening:
		if g.Age >= 14 {
			g.Mode = Playing
			g.Frame = 2
			g.Age = 0
		}
	case Playing:
		for i := 0; i < 2; i++ {
			if click[i]&Start != 0 && g.LevelTicks > 180 && !g.Lost {
				g.Paused = !g.Paused
				if g.Paused {
					g.play("pause", 9)
				}
			}
			if pad[i] != g.OldPad[i] && !g.Paused && !g.Lost {
				if pad[i]&(A|B|C) != 0 && !g.FireHeld[i] {
					g.Tanks[i].Fire = 1
					g.FireHeld[i] = true
				}
				if pad[i]&(A|B|C) == 0 {
					g.FireHeld[i] = false
				}
			}
		}
		if !g.Paused {
			g.update(pad)

		}
		g.LevelTicks++
		g.VisualExplosions = g.Explosions
		g.VisualFloat = g.Float
		g.VisualReady = true
		g.animate()
		g.WaterFrame++
		if g.WaterFrame&63 == 0 {
			g.WaterPhase = 1
		}
		if g.WaterFrame&63 == 31 {
			g.WaterPhase = 2
		}
		if g.Lost {
			if g.Age >= 256 {
				g.beginScore()
			}
		} else {
			g.Age = 0
		}
		if g.Won {
			g.beginScore()
		}
	case Score:
		if g.Age >= 576 || g.Age >= 64 && p1|p2 != 0 {
			if g.Lost {
				for i := 0; i < 2; i++ {
					if g.Tanks[i].Score > g.HighScore {
						g.HighScore = g.Tanks[i].Score
					}
				}
				g.Mode = GameOver
				g.Age = 0
				g.play("game_over", 9)
			} else {
				g.StageNumber++
				g.ClosingFrom = Score
				g.ClosingTo = Intro
				g.Mode = Closing
				g.Age = 0
			}
		}
	case GameOver:
		if g.Age >= 128 {
			g.ReturnToMenu()
		}
	case Editor:
		g.updateEditor(p1, p1^g.OldPad[0])
	}
	g.OldPad = pad
}

func (g *Game) initGame() {
	g.Tanks = [8]Tank{}
	g.Bullets = [16]Bullet{}
	g.Explosions = [16]Explosion{}
	g.VisualReady = false
	g.Tanks[0].Lives = 3
	if g.TwoPlayers {
		g.Tanks[1].Lives = 3
	}
	g.Paused = false
	g.SpeedCounter = 128
	g.Lost = false
	g.Won = false
	g.Float = FloatingScore{}
	g.FireHeld = [2]bool{}
}
func (g *Game) initPlayer(i int) {
	t := &g.Tanks[i]
	if t.Lives <= 0 {
		return
	}
	if t.HP == 0 {
		t.BulletLimit = 1
		t.BulletSpeed = 1
		t.SteelShot = 0
		t.Type = 0
	}
	t.HP = 1
	t.Shield = 256
	t.Direction = 0
	t.Speed = 0
	t.Birth = 64
	t.Bullets = 0
	t.Freeze = 0
	t.Color = i + 1
	t.X = 64 + i*64
	t.Y = 192
	g.VictoryTimer = 0
}
func (g *Game) initLevel() {
	g.Remaining = 20
	g.SpawnDelay = max(74, (35-g.StageNumber%35)*256/35)
	g.SpawnTimer = 0
	g.Won = false
	g.Lost = false
	g.VictoryTimer = 0
	g.LevelTicks = 0
	g.Freeze = 0
	g.BaseArmor = 0
	g.Kills = [2][4]int{}
	g.Bonus = Bonus{}
	g.Float = FloatingScore{}
	g.Bullets = [16]Bullet{}
	g.Explosions = [16]Explosion{}
	g.VisualReady = false
	for i := range g.Tanks {
		g.Tanks[i].Bullets = 0
		g.Tanks[i].Freeze = 0
		g.Tanks[i].Fire = 0
		if i >= 2 {
			g.Tanks[i].HP = 0
		}
	}
	g.initPlayer(0)
	g.initPlayer(1)
	src := g.resources.Maps.Levels[g.StageNumber%35]
	if g.Edited {
		g.Map = g.CustomMap
		g.Edited = false
	} else {
		for y := 0; y < 26; y++ {
			copy(g.Map[y][:], src[y])
		}
	}
	for _, x := range []int{0, 12, 24, 8, 16} {
		y := 0
		if x == 8 || x == 16 {
			y = 24
		}
		for j := 0; j < 2; j++ {
			for k := 0; k < 2; k++ {
				g.Map[y+j][x+k] = 0
			}
		}
	}
	for y := 0; y < 26; y++ {
		for x := 0; x < 26; x++ {
			if g.Map[y][x] == 34 {
				g.Map[y][x] |= 0x8000
			}
		}
	}
	g.Map[24][12] = 252
	g.Map[25][12] = 253
	g.Map[24][13] = 254
	g.Map[25][13] = 255
}
func (g *Game) update(pad [2]uint32) {
	g.SpeedCounter++
	g.SpeedCounter |= 128
	for i := 0; i < 2; i++ {
		if g.Tanks[i].HP == 0 {
			g.initPlayer(i)
		}
	}
	if g.Tanks[0].HP == 0 && g.Tanks[1].HP == 0 {
		g.Lost = true
	}
	for i := range g.Tanks {
		if g.Tanks[i].Fire != 0 {
			g.Tanks[i].Fire = 0
			g.shot(i)
		}
	}
	for i := 0; i < 2; i++ {
		joy := pad[i]
		speed := 3
		if g.Lost {
			joy = 0
			speed = 0
		}
		g.control(i, joy, speed)
	}
	g.updateAI()
	g.updateBullets()
	g.updateMove()
	g.updateBonus()
	g.updateArmor()
}
func (g *Game) control(i int, pad uint32, speed int) {
	t := &g.Tanks[i]
	if t.Birth != 0 || t.HP == 0 {
		return
	}
	if t.Freeze > 0 {
		t.Freeze--
		t.Speed = 0
		return
	}
	t.Speed = 0
	dir := -1
	if pad&Up != 0 {
		dir = 0
	} else if pad&Left != 0 {
		dir = 1
	} else if pad&Down != 0 {
		dir = 2
	} else if pad&Right != 0 {
		dir = 3
	}
	if dir < 0 {
		return
	}
	t.Speed = speed
	t.Direction = dir
	if dir&1 == 0 {
		if t.X&7 > 4 {
			t.X = (t.X >> 3 << 3) + 8
		} else {
			t.X -= t.X & 7
		}
		t.X = min(192, t.X)
	} else {
		if t.Y&7 > 4 {
			t.Y = (t.Y >> 3 << 3) + 8
		} else {
			t.Y -= t.Y & 7
		}
		t.Y = min(192, t.Y)
	}
}
func (g *Game) overlapDistance(i int) int {
	t := g.Tanks[i]
	for j := 7; j >= 0; j-- {
		b := g.Tanks[j]
		if i == j || b.HP == 0 {
			continue
		}
		dx, dy := abs(t.X-b.X), abs(t.Y-b.Y)
		if dx < 16 && dy < 16 {
			return dx + dy
		}
	}
	return 32
}
func (g *Game) updateMove() {
	for i := range g.Tanks {
		t := &g.Tanks[i]
		if t.HP == 0 {
			continue
		}
		if t.Birth > 0 {
			t.Birth--
			continue
		}
		if t.Shield > 0 {
			t.Shield--
		}
		if t.Ice > 0 {
			t.Ice--
		}
		speed := int(g.SpeedCounter) & t.Speed
		if t.Speed == 0 && t.Ice > 0 && i < 2 {
			speed = int(g.SpeedCounter) & 1
		}
		if speed == 0 {
			continue
		}
		before := g.overlapDistance(i)
		dx, dy := directionX[t.Direction], directionY[t.Direction]
		t.X += dx
		t.Y += dy
		if !g.walls(t) {
			t.X -= dx
			t.Y -= dy
			t.Collision++
		}
		if before > g.overlapDistance(i) {
			t.X -= dx
			t.Y -= dy
			t.Collision++
		}
	}
}
func (g *Game) walls(t *Tank) bool {
	x, y := t.X+directionX[t.Direction], t.Y+directionY[t.Direction]
	x1, y1, x2, y2 := x>>3, y>>3, (x+15)>>3, (y+15)>>3
	if x1 < 0 || x1 > 24 || y1 < 0 || y1 > 24 {
		return false
	}
	first := g.tile(x1, y1)
	if first == 33 && t.Speed != 0 {
		t.Ice = 32
	} else if first != 33 {
		t.Ice = 0
	}
	tiles := (first | g.tile(x2, y1) | g.tile(x1, y2) | g.tile(x2, y2)) & 511
	return tiles == 0 || tiles == 33 || tiles == 34 || tiles == 35
}
func (g *Game) tile(x, y int) uint16 {
	if x < 0 || y < 0 || x >= 26 || y >= 26 {
		return 0
	}
	return g.Map[y][x]
}
func (g *Game) setTile(x, y int, v uint16) {
	if x >= 0 && y >= 0 && x < 26 && y < 26 {
		g.Map[y][x] = v
	}
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (g *Game) Save() ([]byte, error) { return json.Marshal(g) }
func (g *Game) Load(b []byte) error {
	var next Game
	if err := json.Unmarshal(b, &next); err != nil {
		return err
	}
	if next.PAL != g.PAL || next.Mode > Closing || next.StageNumber < 0 || next.StageNumber > 65535 {
		return fmt.Errorf("incompatible saved game")
	}
	for _, t := range next.Tanks {
		if t.Direction < 0 || t.Direction > 3 || t.Type < 0 || t.Type > 7 {
			return fmt.Errorf("invalid tank")
		}
	}
	for _, b := range next.Bullets {
		if b.Direction < 0 || b.Direction > 3 || b.Owner < 0 || b.Owner >= 8 {
			return fmt.Errorf("invalid bullet")
		}
	}
	next.resources = g.resources
	*g = next
	return nil
}
