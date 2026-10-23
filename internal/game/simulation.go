package game

func (g *Game) updateAI() {
	count := 0
	for i := 2; i < 8; i++ {
		if g.Tanks[i].HP != 0 {
			count++
		}
	}
	limit := 4
	if g.TwoPlayers {
		limit = 6
	}
	if count == 0 && g.Remaining == 0 {
		g.VictoryTimer++
		if g.VictoryTimer > 256 {
			g.Won = true
		}
		return
	}
	if count < limit && g.Remaining > 0 && g.SpawnTimer == 0 {
		g.spawnEnemy()
	}
	if g.SpawnTimer > 0 {
		g.SpawnTimer--
	}
	if g.Freeze > 0 {
		for i := 2; i < 8; i++ {
			g.Tanks[i].Speed = 0
			g.Tanks[i].Fire = 0
		}
		g.Freeze--
		return
	}
	// Preserve the original restore-after-clock behavior, including its shared
	// speed reset, rather than substituting the behavior of later ROM revisions.
	if g.Tanks[2].Speed == 0 {
		for i := 2; i < 8; i++ {
			g.Tanks[i].Speed = 1
		}
	}
	for i := 2; i < 8; i++ {
		t := &g.Tanks[i]
		if t.Collision == 1 {
			t.FireTimer >>= 1
		}
		if t.Collision > 5 || g.random() < 300 || (t.Collision != 0 && g.random() < 2000) {
			t.Collision = 0
			dir := g.random() & 3
			joy := [4]uint32{Up, Down, Left, Right}[dir]
			g.control(i, joy, t.Speed)
		}
		if t.FireTimer != 0 {
			t.FireTimer--
		} else {
			t.Fire = 1
			t.FireTimer = 64 + int(g.random()&63)
		}
	}
}
func (g *Game) spawnEnemy() {
	kind := 0
	s := g.StageNumber
	switch {
	case s == 0:
	case s == 1:
		kind = int(g.random() % 3)
	case s == 2:
		kind = int(g.random() & 3)
	case s < 10:
		kind = int(g.random() & 3)
		if kind == 0 {
			kind = int(g.random() & 3)
		}
	case s < 20:
		kind = int(g.random() & 3)
		if kind < 2 {
			kind = int(g.random() & 3)
		}
		if kind < 2 {
			kind = int(g.random() & 3)
		}
	default:
		kind = 1 + int(g.random()%3)
		if kind != 2 {
			kind = 1 + int(g.random()%3)
		}
	}
	i := 2
	for ; i < 8; i++ {
		if g.Tanks[i].HP == 0 {
			break
		}
	}
	if i == 8 {
		return
	}
	t := &g.Tanks[i]
	g.Remaining--
	t.BulletLimit = 1
	t.BulletSpeed = 1
	t.HP = 1
	t.Speed = 1
	t.Type = kind + 4
	if kind == 1 {
		t.Speed = 0xaa
	}
	if kind == 2 {
		t.BulletLimit = 2
		t.BulletSpeed = 2
	}
	if kind == 3 {
		t.HP = 4
	}
	t.Y = 0
	t.X = [3]int{0, 96, 192}[g.Remaining%3]
	t.Direction = 2
	t.Bullets = 0
	t.Birth = 64
	t.Color = 3
	t.Bonus = 0
	if g.Remaining&3 == 0 {
		t.Bonus = 1
	}
	g.SpawnTimer = g.SpawnDelay
}
func (g *Game) shot(owner int) {
	t := &g.Tanks[owner]
	if t.Bullets >= t.BulletLimit || t.HP == 0 || t.Birth != 0 {
		return
	}
	for i := range g.Bullets {
		b := &g.Bullets[i]
		if b.Speed != 0 {
			continue
		}
		*b = Bullet{X: t.X, Y: t.Y, Direction: t.Direction, Speed: t.BulletSpeed, Owner: owner}
		if b.X&3 == 0 && t.Direction&1 != 0 {
			b.X++
		}
		if b.Y&3 == 0 && t.Direction&1 == 0 {
			b.Y++
		}
		t.Bullets++
		if owner < 2 {
			g.play("shot", 5)
		}
		return
	}
}
func (g *Game) killBullet(i int, explode bool) {
	b := &g.Bullets[i]
	if b.Speed == 0 {
		return
	}
	b.Speed = 0
	x, y := b.X, b.Y
	if b.Direction == 0 {
		y -= 8
	} else if b.Direction == 1 {
		x -= 8
	}
	if explode {
		g.explosion(2, x, y)
	}
	t := &g.Tanks[b.Owner]
	t.Bullets = max(0, t.Bullets-1)
}
func (g *Game) updateBullets() {
	for i := range g.Bullets {
		b := &g.Bullets[i]
		if b.Speed == 0 {
			continue
		}
		steps := b.Speed
		for n := 0; n < steps && b.Speed != 0; n++ {
			b.X += 2 * directionX[b.Direction]
			b.Y += 2 * directionY[b.Direction]
			g.bulletWalls(i)
			if b.Speed == 0 {
				break
			}
			g.bulletTanks(i)
			if b.Speed == 0 {
				break
			}
			if tile := g.tile(b.X>>3, b.Y>>3); tile >= 252 && tile <= 256 {
				g.killBase()
				g.killBullet(i, true)
			}
			if b.Speed == 0 {
				break
			}
			g.bulletBullets(i)
		}
	}
}
func brickAfterHit(tile uint16, direction, dip int) int {
	if tile == 16 {
		return 16
	}
	if tile > 16 || tile == 0 {
		return -1
	}
	mask := 0
	if direction&1 != 0 {
		mask = 10 >> uint(dip)
	} else {
		mask = 12 >> uint(dip*2)
	}
	if int(tile)&mask == int(tile) {
		return -1
	}
	return int(tile) & mask
}
func (g *Game) bulletWalls(i int) {
	b := &g.Bullets[i]
	x := b.X + 2*directionX[b.Direction]
	y := b.Y + 2*directionY[b.Direction]
	x0, y0 := x>>3, y>>3
	dip := ((x | y) & 7) - 1
	// The cartridge uses an unsigned shift here. A negative value means no
	// brick mask, an observable edge case of firing exactly on a cell boundary.
	if dip < 0 {
		dip = 0x3fffffff
	} else {
		dip >>= 2
	}
	if x0 < 0 || x0 > 24 || y0 < 0 || y0 > 24 {
		g.killBullet(i, true)
		if b.Owner < 2 {
			g.play("impact", 6)
		}
		return
	}
	coords := [2][2]int{{x0, y0}, {x0 + 1, y0}}
	if b.Direction&1 != 0 {
		coords[1] = [2]int{x0, y0 + 1}
	}
	kill := false
	last := -1
	for _, p := range coords {
		after := brickAfterHit(g.tile(p[0], p[1]), b.Direction, dip)
		last = after
		if after < 0 {
			continue
		}
		if after == 16 && g.Tanks[b.Owner].SteelShot != 0 {
			after = 0
		}
		g.setTile(p[0], p[1], uint16(after))
		kill = true
	}
	if kill {
		g.killBullet(i, true)
		if b.Owner < 2 {
			if last == 16 {
				g.play("impact", 6)
			} else {
				g.play("brick", 3)
			}
		}
	}
}
func (g *Game) bulletTanks(i int) {
	b := &g.Bullets[i]
	end := 2
	if b.Owner < 2 {
		end = 8
	}
	for j := 0; j < end; j++ {
		t := &g.Tanks[j]
		if t.HP == 0 || t.Birth != 0 || b.Owner == j {
			continue
		}
		x, y := b.X, b.Y
		if b.Direction == 3 {
			x -= 9
		} else if b.Direction == 2 {
			y -= 9
		}
		if abs(x-t.X) <= 14 && abs(y-t.Y) <= 14 {
			g.killBullet(i, true)
			g.hit(j, b.Owner)
			return
		}
	}
}
func (g *Game) bulletBullets(i int) {
	b := &g.Bullets[i]
	if b.Owner >= 2 {
		return
	}
	for j := 15; j >= 0; j-- {
		o := g.Bullets[j]
		if o.Speed == 0 || o.Owner == b.Owner {
			continue
		}
		dx, dy := b.X-o.X, b.Y-o.Y
		if b.Direction&1 != 0 {
			dy += 7
		} else {
			dx += 7
		}
		if o.Direction&1 != 0 {
			dy -= 7
		} else {
			dx -= 7
		}
		if abs(dx) < 6 && abs(dy) < 6 {
			g.killBullet(i, false)
			g.killBullet(j, false)
		}
	}
}
func (g *Game) hit(victim, killer int) {
	t := &g.Tanks[victim]
	if t.Shield != 0 {
		return
	}
	if victim < 2 && killer < 2 {
		t.Freeze = 256
		g.play("impact", 6)
		return
	}
	if t.Bonus != 0 {
		g.generateBonus()
		t.Bonus = 0
	}
	t.HP--
	if t.HP != 0 {
		g.play("impact", 6)
		return
	}
	if killer < 2 && t.Type >= 4 {
		g.Kills[killer][t.Type-4]++
		g.Float = FloatingScore{t.X, t.Y, 40, t.Type - 4}
	}
	g.Tanks[killer].Score += uint32((t.Type - 3) * 100)
	g.explosion(1, t.X, t.Y)
	if victim < 2 {
		g.play("player_explosion", 7)
	} else {
		g.play("enemy_explosion", 7)
	}
	t.Lives--
}
func (g *Game) killBase() {
	if g.Lost {
		return
	}
	g.Lost = true
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			g.setTile(12+x, 24+y, uint16(156+x*2+y))
		}
	}
	g.explosion(1, 96, 192)
	g.play("player_explosion", 8)
}
func (g *Game) explosion(kind, x, y int) {
	for i := 15; i >= 0; i-- {
		if g.Explosions[i].Kind == 0 {
			g.Explosions[i] = Explosion{X: x, Y: y, Kind: kind}
			return
		}
	}
}
func (g *Game) animate() {
	if g.Float.Timer > 0 && !g.Paused {
		g.Float.Timer--
	}
	for i := range g.Explosions {
		e := &g.Explosions[i]
		if e.Kind == 0 {
			continue
		}
		if (e.Frame > 2 && e.Strobe&8 != 0) || (e.Frame <= 2 && e.Strobe&2 != 0) {
			e.Frame++
			e.Strobe = 0
		}
		if e.Kind == 1 && e.Frame > 4 || e.Kind == 2 && e.Frame > 2 {
			e.Kind = 0
		}
		e.Strobe++
	}
}
func (g *Game) generateBonus() {
	g.Bonus = Bonus{X: int(g.random()%22) * 8, Y: int(g.random()%21) * 8, Kind: int(g.random()&7)%6 + 1}
	g.play("bonus_appears", 9)
}
func (g *Game) updateBonus() {
	if g.Bonus.Kind == 0 {
		return
	}
	for i := 0; i < 2; i++ {
		t := &g.Tanks[i]
		if abs(t.X-g.Bonus.X) >= 16 || abs(t.Y-g.Bonus.Y) >= 16 {
			continue
		}
		switch g.Bonus.Kind {
		case 1:
			t.Shield = 1024
		case 2:
			g.Freeze = 1024
		case 3:
			g.BaseArmor = 1024
		case 4:
			t.Type = min(3, t.Type+1)
			if t.Type == 1 {
				t.BulletSpeed = 2
			}
			if t.Type == 2 {
				t.BulletLimit = 2
			}
			if t.Type == 3 {
				t.SteelShot = 1
			}
		case 5:
			for j := 2; j < 8; j++ {
				e := &g.Tanks[j]
				if e.HP != 0 {
					g.explosion(1, e.X, e.Y)
					e.HP = 0
				}
			}
		case 6:
			t.Lives++
		}
		g.play("bonus_collected", 9)
		g.Float = FloatingScore{g.Bonus.X, g.Bonus.Y, 40, 4}
		g.Bonus.Kind = 0
		break
	}
}
func (g *Game) updateArmor() {
	if g.BaseArmor == 0 {
		return
	}
	g.BaseArmor--
	if g.BaseArmor == 1023 || g.BaseArmor < 256 {
		tile := uint16(16)
		if (g.BaseArmor&31 > 16 && g.BaseArmor < 1004) || g.BaseArmor == 0 {
			tile = 15
		}
		for _, p := range [][2]int{{11, 23}, {11, 24}, {11, 25}, {12, 23}, {13, 23}, {14, 23}, {14, 24}, {14, 25}} {
			g.setTile(p[0], p[1], tile)
		}
	}
}
func (g *Game) beginScore() {
	g.Mode = Score
	g.Age = 0
}

var editorCells = [14][4]uint16{
	{0, 0, 0, 0}, {0, 15, 0, 15}, {0, 0, 15, 15}, {15, 0, 15, 0}, {15, 15, 0, 0}, {15, 15, 15, 15},
	{0, 16, 0, 16}, {0, 0, 16, 16}, {16, 0, 16, 0}, {16, 16, 0, 0}, {16, 16, 16, 16},
	{18, 18, 18, 18}, {34, 34, 34, 34}, {33, 33, 33, 33},
}

func (g *Game) openEditor() {
	g.Mode = Editor
	g.EditorX = 0
	g.EditorY = 0
	g.EditorBrush = 0
	g.EditorMoved = false
	g.Repeat = 0
	if !g.Edited {
		for y := 0; y < 26; y++ {
			copy(g.CustomMap[y][:], g.resources.Maps.Editor[y])
		}
	}
}
func (g *Game) updateEditor(pad, changed uint32) {
	directions := Up | Down | Left | Right
	if pad&directions != 0 {
		g.Repeat++
	} else {
		g.Repeat = 0
	}
	if changed == 0 && !(g.Repeat > 32 && g.Repeat&3 == 0) {
		return
	}
	if changed&directions != 0 {
		g.EditorMoved = true
	}
	if pad&Up != 0 {
		g.EditorY = max(0, g.EditorY-1)
		g.EditorMoved = true
	}
	if pad&Down != 0 {
		g.EditorY = min(12, g.EditorY+1)
		g.EditorMoved = true
	}
	if pad&Left != 0 {
		g.EditorX = max(0, g.EditorX-1)
		g.EditorMoved = true
	}
	if pad&Right != 0 {
		g.EditorX = min(12, g.EditorX+1)
		g.EditorMoved = true
	}
	if pad&Start != 0 {
		g.Edited = true
		g.ReturnToMenu()
		return
	}
	if pad&(B|C) != 0 {
		if !g.EditorMoved {
			if pad&B != 0 {
				g.EditorBrush = (g.EditorBrush + 1) % len(editorCells)
			}
			if pad&C != 0 {
				g.EditorBrush = (g.EditorBrush + len(editorCells) - 1) % len(editorCells)
			}
		}
		cells := editorCells[g.EditorBrush]
		x, y := g.EditorX*2, g.EditorY*2
		g.CustomMap[y][x] = cells[0]
		g.CustomMap[y][x+1] = cells[1]
		g.CustomMap[y+1][x] = cells[2]
		g.CustomMap[y+1][x+1] = cells[3]
		g.EditorMoved = false
	}
}
