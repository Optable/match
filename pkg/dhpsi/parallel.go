package dhpsi

// derive & multiply operation
// with a completion. operates on
// any lenght of string
type dmOp struct {
	p []byte
	f func([EncodedLen]byte)
}

// multiply operation
// with a completion. operates on the
//
type mOp struct {
	p [EncodedLen]byte
	f func([EncodedLen]byte)
}

var (
	deriveMultiplyBus chan<- dmOp
	multiplyBus       chan<- mOp
)

func init() {

}
