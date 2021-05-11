package dhpsi

import "bufio"

// ReadLine blocks until a whole line can be read or
// r returns an error. Cannot read more than n bytes
func SafeReadLine(r *bufio.Reader) (line []byte, err error) {
	line, err = r.ReadBytes('\n')
	if len(line) > 1 {
		// strip the \n
		line = line[:len(line)-2]
	}
	return
}
