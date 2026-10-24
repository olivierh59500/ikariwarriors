package game

// These positions were learned from ordinary play. The search still requires
// a reachable path, a current weapon trajectory and a fresh safety comparison.
type demoExpertAnchor struct {
	row, column int
	inTank      bool
	x, y        float64
}

var demoExpertAnchors = [...]demoExpertAnchor{
	{308, 5, false, 104, 5014},
	{308, 5, false, 104, 5016},
	{406, 5, false, 114, 6582},
	{406, 5, false, 114, 6584},
	{406, 5, true, 96, 6582},
	{406, 5, true, 96, 6584},
	{433, 5, true, 96, 7016},
	{433, 5, true, 96, 7018},
	{485, 5, false, 89, 7848},
	{485, 5, false, 126, 7846},
	{485, 5, false, 126, 7848},
	{620, 8, true, 144, 10008},
	{656, 5, false, 120, 10581},
	{656, 5, false, 120, 10583},
	{656, 5, false, 126, 10582},
	{656, 5, false, 126, 10584},
	{656, 5, true, 126, 10584},
	{667, 5, false, 104, 10757},
	{667, 5, false, 104, 10759},
	{667, 5, false, 194, 10742},
	{667, 5, true, 134, 10760},
	{748, 5, false, 98, 12054},
	{748, 5, false, 98, 12056},
	{748, 5, true, 98, 12056},
	{771, 5, false, 98, 12422},
	{771, 5, false, 98, 12424},
	{771, 5, true, 98, 12424},
	{803, 4, false, 139, 12926},
	{803, 4, false, 139, 12928},
	{803, 4, false, 139, 12930},
	{803, 4, false, 176, 12906},
	{803, 4, true, 174, 12840},
	{831, 0, true, 32, 13384},
	{831, 0, true, 46, 13382},
	{831, 0, true, 46, 13384},
}

func (g *Game) hasDemoExpertAnchors(final bool) bool {
	if final {
		return false
	}
	for _, p := range demoExpertAnchors {
		if p.row == g.demoSceneryRow && p.column == g.demoSceneryCol && p.inTank == g.player.inTank {
			return true
		}
	}
	return false
}
func (g *Game) atDemoExpertAnchor(x, y float64) bool {
	for _, p := range demoExpertAnchors {
		if p.row == g.demoSceneryRow && p.column == g.demoSceneryCol && p.inTank == g.player.inTank && p.x == x && p.y == y {
			return true
		}
	}
	return false
}
func (g *Game) demoAnchorReady(x, y float64, aim int) bool {
	future := *g
	future.player.x, future.player.y = x, y
	if !future.demoForecastActive {
		future.prepareDemoForecast()
	}
	target := navPoint{x, y}
	_, normal := future.demoChooseMove(target)
	attack := playerControls{}
	if future.player.direction != aim {
		attack.x, attack.y = directionVector(aim)
	}
	attack.fire, attack.action = future.player.inTank, !future.player.inTank
	_, quality := future.demoPlanAttack(target, &attack, aim)
	return quality > demoBlockedScore && quality >= normal-50
}

// A secondary position is considered only after the usual positions fail.
// It still requires the same reachable path, trajectory and safety checks.
var demoFallbackAnchors = [...]demoExpertAnchor{
	{485, 5, false, 154, 7848},
	{667, 5, true, 170, 10740},
	{667, 5, true, 170, 10742},
	{656, 5, true, 116, 10584},
	{308, 5, false, 103, 5014},
	{308, 5, false, 103, 5016},
	{803, 4, false, 170, 12880},
	{308, 5, false, 103, 5013},
	{308, 5, false, 103, 5015},
}

func (g *Game) hasDemoFallbackAnchors(final bool) bool {
	if final {
		return false
	}
	for _, p := range demoFallbackAnchors {
		if p.row == g.demoSceneryRow && p.column == g.demoSceneryCol && p.inTank == g.player.inTank {
			return true
		}
	}
	return false
}

func (g *Game) atDemoFallbackAnchor(x, y float64) bool {
	for _, p := range demoFallbackAnchors {
		if p.row == g.demoSceneryRow && p.column == g.demoSceneryCol && p.inTank == g.player.inTank && p.x == x && p.y == y {
			return true
		}
	}
	return false
}
