package kkrtpsi

import "math"

func findK(n int64) int {
	logSize := uint8(math.Log2(float64(n)))

	switch {
	case logSize > 0 && logSize <= 8:
		return 424
	case logSize > 8 && logSize <= 12:
		return 432
	case logSize > 12 && logSize <= 16:
		return 440
	case logSize > 16 && logSize <= 20:
		return 448
	case logSize > 20 && logSize <= 24:
		return 448
	default:
		return 128
	}
}
