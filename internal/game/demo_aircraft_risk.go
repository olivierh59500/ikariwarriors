package game

import "math"

// Fragments from one helicopter section can destroy the other while the
// radial pool is busy, creating an immediate hostile quad. Treat that visible
// chain reaction as a reason to keep clear of the surviving sections, rather
// than assuming that only the first predicted explosion can become dangerous.
func (g *Game) demoAircraftBurstRisk() bool {
	if !g.aircraft.alive[0] && !g.aircraft.alive[3] {
		return false
	}
	near := func(x, y float64) bool {
		for _, index := range [...]int{0, 3} {
			if !g.aircraft.alive[index] {
				continue
			}
			part := g.aircraft.part(index)
			if math.Abs(part.x-x) <= 128 && math.Abs(part.y-y) <= 128 {
				return true
			}
		}
		return false
	}
	if g.burst.active {
		var center navPoint
		for _, part := range g.burst.parts {
			center.x += part.x / 8
			center.y += part.y / 8
		}
		if near(center.x, center.y) {
			return true
		}
	}
	for _, request := range g.burstQueue {
		if near(request.x, request.y) {
			return true
		}
	}
	for _, burst := range g.demoTargetBursts[:g.demoBurstCount] {
		if burst.firstMove <= demoHorizon+8 && near(burst.x, burst.y) {
			return true
		}
	}
	for _, shot := range g.demoShots() {
		if first, fragments := g.demoFragmentStart(shot); fragments && first <= demoHorizon+8 {
			if near(shot.x+shot.vx*float64(max(0, shot.life)), shot.y+shot.vy*float64(max(0, shot.life))) {
				return true
			}
		}
	}
	return false
}

func (g *Game) demoAircraftStandoffCost(x, y float64, tick int) float64 {
	if !g.demoAircraftRisk {
		return 0
	}
	cost := 0.0
	for _, index := range [...]int{0, 3} {
		if !g.aircraft.alive[index] {
			continue
		}
		part := g.aircraft.part(index)
		travel := 0.0
		if g.aircraft.flying {
			travel = float64(tick * 2)
		}
		// A front-section loss can stop the tail during this interval.
		// Retain both its present and still-flying possible positions.
		depth := min(x-part.x+64, part.x+72-x, y-part.y+72, part.y+64+travel-y)
		if depth > 0 {
			cost += 8 * depth * depth
		}
	}
	return cost
}
