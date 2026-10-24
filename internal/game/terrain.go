package game

// Composite scenery is stored as runs of row codes, rather than independent
// destructible tiles. $11776 finds the left edge, then the top row, and clears
// the rectangle described by $13132. Its row/column counts are DBF counts.
func (g *Game) destroyTileAt(x, y float64) bool {
	row, column, size, ok := g.sceneryBounds(x, y)
	if !ok {
		return false
	}
	for r := row; r < min(mapRows, row+size.rows); r++ {
		for c := column; c < min(mapColumns, column+size.columns); c++ {
			g.destroyed[r][c] = true
		}
	}
	// The original paints its ground BOB at the retained two-pixel probe
	// phase. Logical cleared cells and the painted border have distinct origins.
	// Forecasts copy Game values. Give each appended history its own backing
	// array so a later live clear cannot alter a retained predicted history.
	g.sceneryPaint = append(g.sceneryPaint[:len(g.sceneryPaint):len(g.sceneryPaint)], sceneryGroundPatch{
		row: row, column: column, rows: size.rows, columns: size.columns,
		yOffset: -(int(y) & 1), clipTop: int(g.cameraBackY) - 32,
	})
	return true
}

func (g *Game) sceneryBounds(x, y float64) (int, int, compositeSize, bool) {
	if x < 0 || y < 0 {
		return 0, 0, compositeSize{}, false
	}
	column, row := int(x)/tileSize, int(y+15)/tileSize
	if column < 0 || column >= mapColumns || row < 0 || row >= mapRows {
		return 0, 0, compositeSize{}, false
	}
	code := g.resources.levelCodes[row][column]
	if code <= 0xec || g.destroyed[row][column] {
		return 0, 0, compositeSize{}, false
	}
	for column > 0 && g.resources.levelCodes[row][column-1] == code && !g.destroyed[row][column-1] {
		column--
	}
	for row > 0 && g.resources.levelCodes[row-1][column] > 0xec && !g.destroyed[row-1][column] {
		row--
	}
	size := g.resources.composites[g.resources.levelCodes[row][column]-0xed]
	return row, column, size, true
}

func (g *Game) damageTerrain(x, y float64, owner int) bool {
	effect, hasEffect := g.sceneryBurstAt(x, y)
	if !g.destroyTileAt(x, y) {
		return false
	}
	if hasEffect {
		g.startSceneryBurst(effect)
	}
	// The original awards one thousand points once for the whole object.
	if owner == 1 {
		g.score2 += 1000
	} else {
		g.score += 1000
	}
	return true
}

func (g *Game) displayTile(row, column int) int {
	if g.destroyed[row][column] {
		// The original writes raw code zero, whose tile is zero in both banks.
		return 0
	}
	return int(g.resources.level[row][column])
}

// Explosive scenery also detonates on contact. Ordinary infantry bullets stop
// against it; walking into it is dangerous, even though the explosion clears
// the way. This is the player-only branch in $10eae/$10edc.
// The original water probes reject river codes before the infantry passable
// table when a player is inside a tank, including during the escape interval.
func tankWaterCode(code byte) bool { return code >= 139 && code <= 158 }

func (g *Game) playerTerrainPassable(value *actor, x, y float64) bool {
	if value.inTank && tankWaterCode(g.terrainCode(x, y)) {
		return false
	}
	owner := 0
	if value == &g.player2 {
		owner = 1
	}
	if g.damageTerrain(x, y, owner) {
		g.hitPlayer(owner)
		return false
	}
	return !g.solidAt(x, y)
}
