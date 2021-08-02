package util

import "math/rand"

// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
func Transpose(matrix [][]uint8) [][]uint8 {
	n := len(matrix)
	tr := make([][]uint8, len(matrix[0]))

	for row := range tr {
		tr[row] = make([]uint8, n)
		for col := range tr[row] {
			tr[row][col] = matrix[col][row]
		}
	}
	return tr
}

// Transpose3D returns the transpose of a 3D slices of uint8
// from (m x 2 x k) to (k x 2 x m)
func Transpose3D(matrix [][2][]uint8) [][2][]uint8 {
	n := len(matrix)
	tr := make([][2][]uint8, len(matrix[0][0]))

	for row := range tr {
		for b := range tr[row] {
			tr[row][b] = make([]uint8, n)
			for col := range tr[row][b] {
				tr[row][b][col] = matrix[col][b][row]
			}
		}
	}
	return tr
}

// SampleRandomBitMatrix fills each entry in the given 2D slices of uint8
// with pseudorandom bit values
func SampleRandomBitMatrix(r *rand.Rand, m, k int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, m)
	for row := range matrix {
		matrix[row] = make([]uint8, k)
	}

	for row := range matrix {
		if err := SampleBitSlice(r, matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

// SampleBitSlice returns a slice of uint8 of pseudorandom bits
func SampleBitSlice(prng *rand.Rand, b []uint8) (err error) {
	// read up to len(b) pseudorandom bits
	t := make([]byte, len(b)/8)
	if _, err = prng.Read(t); err != nil {
		return nil
	}

	// extract all bits into b
	extractBytesToBits(t, b)

	return nil
}

func extractBytesToBits(t, b []byte) {
	var i int
	for _, _byte := range t {
		b[i] = uint8(_byte & 0x01)
		b[i+1] = uint8((_byte >> 1) & 0x01)
		b[i+2] = uint8((_byte >> 2) & 0x01)
		b[i+3] = uint8((_byte >> 3) & 0x01)
		b[i+4] = uint8((_byte >> 4) & 0x01)
		b[i+5] = uint8((_byte >> 5) & 0x01)
		b[i+6] = uint8((_byte >> 6) & 0x01)
		b[i+7] = uint8((_byte >> 7) & 0x01)
		i += 8
	}
}
