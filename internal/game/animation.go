package game

// walkAnimation translates $fb2a/$fb82. Torso and leg counters advance
// independently, every other gameplay tick, and keep their phase when the
// joystick changes direction. Only their delay bytes are reset on a turn.
type walkAnimation struct {
	step  int
	delay int
	frame int
}

func (a *walkAnimation) advance(sequence []int) {
	if a.delay != 0 {
		a.delay--
		return
	}
	a.delay = 1
	a.step++
	if a.step >= len(sequence) {
		a.step = 0
	}
	a.frame = sequence[a.step]
}

func (a *actor) face(direction int) {
	if a.direction != direction {
		a.upper.delay, a.lower.delay = 0, 0
	}
	a.direction = direction
}

func (a *actor) animateWalk(player bool) {
	direction := a.direction & 7
	if player {
		advancePlayerWalk(&a.upper, &a.lower, 0, direction, a.wet, false)
	} else {
		a.upper.advance(enemyActorSequences[direction])
		a.lower.advance(enemyPartSequences[direction])
	}
}

func digitalAxis(value float64) float64 {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return 0
}

// $c738 launches the held-button grenade on the eleventh tick, then repeats
// every ten ticks until the fire button is released.
func heldGrenade(duration int) bool {
	return duration >= 11 && (duration-11)%10 == 0
}

func grenadeButton(duration int) bool {
	return duration >= 1 && (duration-1)%10 == 0
}

// A digital joystick has a stable cardinal sector on either side of each
// axis. Keep that sector for touch and analog sticks instead of interpreting
// every small cross-axis deviation as a diagonal command.
func joystickAxes(dx, dy float64) (float64, float64) {
	ax, ay := dx, dy
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	if max(ax, ay) < 0.18 {
		return 0, 0
	}
	const diagonalBoundary = 0.4142135623730951
	if ax < ay*diagonalBoundary {
		dx = 0
	}
	if ay < ax*diagonalBoundary {
		dy = 0
	}
	return digitalAxis(dx), digitalAxis(dy)
}
