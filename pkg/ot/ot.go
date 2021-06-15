package ot

// each base OT has the following 2 methods
type Ot interface {
	Send()
	Receive()
}

// Sender is a generic OT sender type
// that has a slice of pairs of messages
type Sender struct {
	// arrays of m_0, m_1
	messages [][2]string
	// channel to send encrypted msgs
	c chan []byte
}

// Receiver is a generic OT receiver type
// that has a slice of choic bits and a slice
// of received messages
type Receiver struct {
	// choice bits
	choices []uint8
	//messages
	messages []string
	// channel to send choice bits and receive messages
	c chan []byte
}

// A base OT has a sender and a receiver
type BaseOt struct {
	Sender
	Receiver
}

// NewSender returns a Sender struct
func NewSender(messages [][2]string) Sender {
	return Sender{
		messages: messages,
		c:        make(chan []byte),
	}
}

// NewReceiver returns a Receiver struct
func NewReceiver(choices []uint8) Receiver {
	return Receiver{
		choices: choices,
		// receives # of choice bit messages
		messages: make([]string, len(choices)),
		c:        make(chan []byte),
	}
}

// InitBaseOT returns a BaseOT struct with number of baseOT configured.
func InitBaseOt(baseCount int) *BaseOt {
	msgs := make([][2]string, baseCount)
	choices := make([]uint8, baseCount)
	return &BaseOt{
		NewSender(msgs),
		NewReceiver(choices),
	}
}
