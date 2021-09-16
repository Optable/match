package kkrtpsi

// findK returns the number of base OT for OPRF
// these numbers are from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
// by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016.
// Reference:	http://dx.doi.org/10.1145/2976749.2978381 (KKRT)
func findBitsetK(n int64) int {
	switch {
	// 2^8
	case n > 0 && n <= 256:
		return 320
	// 2^12
	case n > 256 && n <= 4096:
		return 384
	// 2^16
	case n > 4096 && n <= 65536:
		return 448
	// 2^20
	case n > 65536:
		return 512
	default:
		return 128
	}
}
