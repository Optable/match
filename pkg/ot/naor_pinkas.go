package ot

type naorPinkas struct {
	baseCount int
}

func NewNaorPinkas(baseCount int) (*naorPinkas, error) {
	return &naorPinkas{baseCount}, nil
}

func (n *naorPinkas) Send(messages [][2]string, c chan []byte) error {
	return nil
}

func (n *naorPinkas) Receive(choices []uint8, messages []string, c chan []byte) error {
	return nil
}
