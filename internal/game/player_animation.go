package game

// The original walking tables end with a -1 word. Restarted players can
// retain a later phase from another pose, so the following words remain
// meaningful animation data until the next sentinel is reached.
var playerAnimationWords = [...]int16{
	0, 320, 640, 320, -1, 960, -1, -1, -1, -1, 7360, 7680,
	7360, 7040, -1, 8000, -1, -1, -1, -1, 1280, 1600, 1920, 1600,
	-1, 8320, 8640, 8960, 8640, -1, 3200, 3520, 3840, 3520, -1, 10240,
	10560, 10880, 10560, -1, 4160, 4480, 4800, 4480, -1, 11200, 11520, 11840,
	11520, -1, 12160, 12480, 12800, 12480, -1, 5120, 5440, 5760, 5440, -1,
	6080, 6400, 6720, 6400, -1, 13120, 13440, 13760, 13440, -1, 2240, 2560,
	2880, 2560, -1, 9280, 9600, 9920, 9600, -1, 2560, 6400, 320, 3520,
	1600, 3520, 320, 6400, -1, 9600, 12480, 8000, 11520, 8640, 11520, 8000,
	12480, -1,
}

var playerWalkingTables = [2][8]int{
	{0x12e92, 0x12e74, 0x12e10, 0x12e56, 0x12e2e, 0x12e42, 0x12e06, 0x12e7e},
	{0x12e9c, 0x12e6a, 0x12e24, 0x12e60, 0x12e38, 0x12e4c, 0x12e1a, 0x12e88},
}

var playerLegTables = [8]int{0x13066, 0x13090, 0x13058, 0x13082, 0x13066, 0x13074, 0x1304a, 0x1309e}

func walkingAnimationWord(address int) (int, bool) {
	if address >= 0x12e06 && address < 0x12eca {
		return int(playerAnimationWords[(address-0x12e06)/2]), true
	}
	if address >= 0x12eca && address < 0x12eca+2*len(infantryAnimationWords) {
		return int(infantryAnimationWords[(address-0x12eca)/2]), true
	}
	return 0, false
}

func advancePlayerImage(value *walkAnimation, address, stride, bank int) bool {
	if value.delay != 0 {
		value.delay--
		return false
	}
	value.delay = 1
	value.step = (value.step + 1) & 255
	offset, known := walkingAnimationWord(address + 2*int(int8(value.step)))
	if !known || offset == -1 {
		// Normal retained walking, waving and falling phases stay within the
		// recovered static words. Unknown offsets restart at the known pose.
		value.step = 0
		offset, _ = walkingAnimationWord(address)
	}
	value.frame = infantryFrameForOffset(offset-bank, stride)
	return true
}

func advancePlayerWalk(upper, lower *walkAnimation, player, direction int, wet, locked bool) {
	direction &= 7
	if advancePlayerImage(lower, playerLegTables[direction], overlayBobBytes, 0) && wet {
		offset := infantryImageOffset(lower.frame, overlayBobBytes)
		lower.frame = -3
		if offset&0x40 != 0 {
			lower.frame = -2
		}
	}
	if !locked {
		advancePlayerImage(upper, playerWalkingTables[player][direction], actorBobBytes, player*22*actorBobBytes)
	}
}
