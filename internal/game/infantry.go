package game

// The original retains each slot's animation phases and leg image after it
// disappears. A later spawn assigns its torso without resetting those phases.
type infantryPose struct {
	upper, lower walkAnimation
}

func (g *Game) rememberInfantryAnimation(value actor) {
	if value.slot >= 5 && value.slot <= 18 && (value.kind == 0 || value.kind == 5) {
		g.infantryPoses[value.slot] = infantryPose{value.upper, value.lower}
	}
}

// The final encounter and ending overwrite these slots with complete wide
// bodies. Their original initializers clear the leg images while retaining
// both parts' independent animation phase and delay counters.
func (g *Game) clearInfantryLegImages(first, last int) {
	for slot := first; slot <= last; slot++ {
		g.infantryPoses[slot].lower.frame = -1
	}
}

// Negative leg frames retain the two shared water images, independently of
// the soldier's normal or red art bank. A delay-only tick keeps its image.
func (value actor) infantryArtAddresses() (upper, lower int) {
	upper, lower = 0x4cb88+infantryImageOffset(value.upper.frame, actorBobBytes), 0x50288+infantryImageOffset(value.lower.frame, overlayBobBytes)
	if value.kind == 5 {
		upper, lower = 0x51e08+infantryImageOffset(value.upper.frame, actorBobBytes), 0x54608+infantryImageOffset(value.lower.frame, overlayBobBytes)
	}
	switch value.lower.frame {
	case -1:
		lower = 0
	case -2:
		lower = 0x4a1a8
	case -3:
		lower = 0x4a248
	}
	return upper, lower
}

func (g *Game) animateInfantryLegs(value *actor, sequence []int, wet bool) {
	assigning := value.lower.delay == 0
	value.lower.advance(sequence)
	if assigning && wet {
		if value.lower.frame*overlayBobBytes&0x40 != 0 {
			value.lower.frame = -2
		} else {
			value.lower.frame = -3
		}
	}
}

// $a7e0 uses a downward entry timer, then reverses both diagonal components
// on the upward path. A blocked timed step rolls back; an untimed step keeps
// its attempted position and changes the next path. Both still animate.
func (g *Game) advanceInfantryMarch(value *actor) (direction int, wet, moving bool) {
	if value.marchTime == 0 && value.shots > 0 {
		return 0, false, false
	}
	timed := value.marchTime > 0
	dx, dy := 0.0, 2.0
	if value.marchX == 1 {
		dx = -2
	} else if value.marchX != 0 {
		dx = 2
	}
	if !timed && value.marchY == 1 {
		dx, dy = -dx, -dy
	}
	direction = directionForVector(dx, dy)
	previousX, previousY := value.x, value.y
	value.x, value.y = value.x+dx, value.y+dy
	if timed {
		value.marchTime--
	}
	code := g.terrainCode(value.x, value.y)
	wet = code == 157 || code == 158
	if g.solidAt(value.x, value.y) {
		if timed {
			value.x, value.y, value.marchTime = previousX, previousY, 0
		} else {
			value.marchY, value.shots = 1, 3
			if dy < 0 {
				value.marchY = 0
			}
		}
	}
	return direction, wet, true
}

// A retained phase can select a half-frame torso offset from the following
// original table. Keep ordinary frame indices while preserving those exact
// byte offsets without adding machine state to the actor.
const infantryRawFrame = 1 << 20

func infantryImageOffset(frame, stride int) int {
	if frame >= infantryRawFrame-32768 {
		return frame - infantryRawFrame
	}
	return frame * stride
}

func infantryFrameForOffset(offset, stride int) int {
	if offset >= 0 && offset%stride == 0 {
		return offset / stride
	}
	return infantryRawFrame + offset
}

var infantryWalkTables = [...][2]int{
	{0x12fca, 0x130ca}, {0x12ffc, 0x130e8},
	{0x12fe8, 0x130b6}, {0x13010, 0x130de},
	{0x12fd4, 0x130c0}, {0x13006, 0x130d4},
	{0x12fde, 0x130ac}, {0x12ff2, 0x130f2},
}

// $fb2a/$fb82 wrap on the selected -1 word, rather than a slice boundary.
// F1 can retain unfinished death phases beyond a walking table's first four
// entries, and both parts keep their independent delays during that restart.
func advanceInfantryImage(value *walkAnimation, address, stride int) {
	if value.delay != 0 {
		value.delay--
		return
	}
	value.delay = 1
	value.step = (value.step + 1) & 255
	index := (address-0x12eca)/2 + int(int8(value.step))
	if index < 0 || index >= len(infantryAnimationWords) {
		// An extreme retained phase can leave the static animation words and
		// reach mutable source input state. That unobserved overrun is not an
		// animation asset; restart this part within its known original table.
		value.step = 0
		index = (address - 0x12eca) / 2
	}
	offset := int(infantryAnimationWords[index])
	if offset == -1 {
		value.step = 0
		offset = int(infantryAnimationWords[(address-0x12eca)/2])
	}
	value.frame = infantryFrameForOffset(offset, stride)
}

func (g *Game) animateInfantryWalk(value *actor, direction int, wet bool) {
	table := infantryWalkTables[direction&7]
	assigning := value.lower.delay == 0
	advanceInfantryImage(&value.lower, table[1], overlayBobBytes)
	if assigning && wet {
		if infantryImageOffset(value.lower.frame, overlayBobBytes)&0x40 != 0 {
			value.lower.frame = -2
		} else {
			value.lower.frame = -3
		}
	}
	advanceInfantryImage(&value.upper, table[0], actorBobBytes)
}

// Original static words surrounding the walking tables through $1317c.
// Mutable selector, fire-hold and target-position words are excluded.
var infantryAnimationWords = [...]int16{
	2880, 1920, 960, 0, -1, 0, 960, 1920, 2880, -1, 7680, 8640,
	9600, 10560, -1, 10560, 9600, 8640, 7680, -1, 5760, 6720, -1, -1,
	-1, 6720, 5760, -1, -1, -1, 3840, 4800, -1, -1, -1, 4800,
	3840, -1, -1, -1, 12480, 12960, 13440, 13920, -1, 7840, 8480, 9120,
	9760, -1, 1600, 1760, -1, 2240, 2400, -1, 1920, 2080, -1, 11520,
	11840, 12160, 12480, 12800, 13120, 13440, 13760, -1, 1760, 2720, 480, 4640,
	2400, 3840, 960, 3360, -1, 7680, 8000, 8320, 8640, 8960, 9280, 9600,
	9920, -1, 0, 320, 640, 960, 1280, 1600, 1920, 2240, 2560, 2880,
	3200, 3520, 3200, -1, 3840, 4160, 4480, 4800, 5120, 5440, 5760, 6080,
	6400, 6720, 7040, 7360, 7040, -1, 0, 160, 320, 480, 640, 800,
	960, 1120, 1280, 1440, 1600, 1600, 1600, -1, 2880, 3200, 3520, 3200,
	-1, 1920, 2240, 2560, 2240, -1, 0, 320, 640, 320, -1, 960,
	1280, 1600, 1280, -1, 6720, 7040, 7360, 7040, -1, 5760, 6080, 6400,
	6080, -1, 3840, 4160, 4480, 4160, -1, 4800, 5120, 5440, 5120, -1,
	3040, 3200, -1, 3040, 3360, -1, 3040, 3520, -1, 3840, 3680, -1,
	3040, 4000, -1, 3040, 4160, -1, 3040, 4320, -1, 3040, 4480, -1,
	0, 160, 320, 480, 640, 800, -1, 960, 1120, 1280, 1440, 1600,
	1760, -1, 1920, 2080, 2240, 2400, 2560, 2720, -1, 2880, 3040, 3200,
	3360, 3520, 3680, -1, 3840, 4000, 4160, 4320, 4480, 4640, -1, 4800,
	4960, 5120, 5280, 5440, 5600, -1, 5760, 5920, 6080, 6240, 6400, 6560,
	-1, 0, 160, 320, 480, -1, 640, 800, 960, 1120, -1, 1280,
	1440, 1600, 1760, -1, 1920, 2080, 2240, 2400, -1, 2560, 2720, 2880,
	3040, -1, 3200, 3360, 3520, 3680, -1, 3840, 4000, 4160, 4320, -1,
	4480, 4640, 4800, 4960, -1, 960, 3840, 3360, -1, 0, 6240, 2400,
	-1, 2880, 5760, 5280, -1, 1440, 4320, 1920, -1, 480, 6720, 4800,
	-1, 960, 0, -1, 1440, 2400, 3360, -1, 3, 2, 3, 2,
	3, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 4,
	2, 4, 2, 4, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 2, 3, 2, 3, 2, 3, 2, 3, 2,
}
