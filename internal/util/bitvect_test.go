package util

var uintBlock = SampleRandomBlock(prng, 1000)
var randomBlock = From(uintBlock[:512])

/*
func TestTranspose(t *testing.T) {
	tr := randomBlock.Transpose()

	for

}
*/
/*
func BenchmarkXorBytes(b *testing.B) {
	a := make([]byte, 10000)
	SampleBitSlice(prng, a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		XorBytes(a, a)
	}
}
*/
