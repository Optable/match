package util

import (
	"bufio"
	"io"
	"log"
)

// ReadLine blocks until a whole line can be read or
// r returns an error.
//  TODO: Cannot read more than n bytes
// ***warning: expects lines to be \r\n separated***
func SafeReadLine(r *bufio.Reader) (line []byte, err error) {
	line, err = r.ReadBytes('\n')
	if len(line) > 1 {
		// strip the \r\n
		line = line[:len(line)-2]
	}
	return
}

// Exhaust all the identifiers in r,
// The format of an indentifier is string\r\n
func Exhaust(n int64, r io.Reader) <-chan []byte {
	// make the output channel
	var identifiers = make(chan []byte)
	// wrap r in a bufio reader
	src := bufio.NewReader(r)
	go func() {
		defer close(identifiers)
		for i := int64(0); i < n; i++ {
			identifier, err := SafeReadLine(src)
			if len(identifier) != 0 {
				identifiers <- identifier
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("error reading identifiers: %v", err)
				}
				return
			}
		}
	}()

	return identifiers
}
