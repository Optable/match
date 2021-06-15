package ot

// each base OT has the following 2 methods
type Ot interface {
	Send()
	Receive()
}

type Sender struct {
	// arrays of m_0, m_1
	messages [][2]string
	// channel to send encrypted msgs
	c chan []byte
}

type Receiver struct {
	// choice bits
	choices []uint8
	//messages
	messages []string
	// channel to send choice bits and receive messages
	c chan []byte
}

func NewSender(messages [][2]string) *Sender {
	return &Sender{
		messages: messages,
		c:        make(chan []byte),
	}
}

func NewReceiver(choices []uint8) *Receiver {
	return &Receiver{
		choices: choices,
		// receives # of choice bit messages
		messages: make([]string, len(choices)),
		c:        make(chan []byte),
	}
}
