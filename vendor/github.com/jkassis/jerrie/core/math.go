package core

// MaxInt returns the maximum of a list of ints
func MaxInt(ints ...int) int {
	winner := ints[0]
	for i := 1; i < len(ints); i++ {
		if ints[i] > winner {
			winner = ints[i]
		}
	}
	return winner
}

// MaxInt8 returns the maximum of a list of int8s
func MaxInt8(ints ...int8) int8 {
	winner := ints[0]
	for i := 1; i < len(ints); i++ {
		if ints[i] > winner {
			winner = ints[i]
		}
	}
	return winner
}

// MaxInt16 returns the maximum of a list of int16s
func MaxInt16(ints ...int16) int16 {
	winner := ints[0]
	for i := 1; i < len(ints); i++ {
		if ints[i] > winner {
			winner = ints[i]
		}
	}
	return winner
}

// MaxInt32 returns the maximum of a list of int32s
func MaxInt32(ints ...int32) int32 {
	winner := ints[0]
	for i := 1; i < len(ints); i++ {
		if ints[i] > winner {
			winner = ints[i]
		}
	}
	return winner
}

// MaxInt64 returns the maximum of a list of int64s
func MaxInt64(ints ...int64) int64 {
	winner := ints[0]
	for i := 1; i < len(ints); i++ {
		if ints[i] > winner {
			winner = ints[i]
		}
	}
	return winner
}
