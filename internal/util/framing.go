package util

import (
	"bufio"
	"io"
	"log"
)

// ReadLine blocks until a whole line can be read or
// r returns an error.
//  TODO: Cannot read more than n bytes
// ***warning: expects lines to be \n separated***
func SafeReadLine(r *bufio.Reader) (line []byte, err error) {
	line, err = r.ReadBytes('\n')
	if len(line) > 1 {
		// strip the \n
		line = line[:len(line)-1]
	}
	return
}

// Exhaust all the identifiers in r,
// The format of an indentifier is string\n
func Exhaust(n int64, r io.Reader) <-chan []byte {
	// make the output channel
	var identifiers = make(chan []byte)
	// wrap r in a bufio reader
	src := bufio.NewScanner(r)
	src.Buffer(make([]byte, 64*1024), 64*1024)
	go func() {
		defer close(identifiers)
		for i := int64(0); i < n; i++ {
			if !src.Scan() {
				if src.Err() != nil {
					log.Printf("error reading identifiers: %v", src.Err())
				}
				return
			}

			identifier := src.Bytes()
			if len(identifier) != 0 {
				identifiers <- identifier
			}
		}
	}()

	return identifiers
}

func Exhaust2(n int64, r io.Reader) <-chan []byte {
	// make the output channel
	var identifiers = make(chan []byte)
	// wrap r in a bufio reader
	src := bufio.NewScanner(r)
	go func() {
		defer close(identifiers)
		for src.Scan() {
			identifiers <- src.Bytes()
		}
		if err := src.Err(); err != nil {
			log.Printf("error reading identifiers: %v", err)
		}
	}()

	return identifiers
}
