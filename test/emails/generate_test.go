package emails

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

const (
	Cardinality       = 100000
	CommonCardinality = Cardinality / 10
)

func initDataSource(common []byte) *bufio.Reader {
	// get an io pipe to read results
	i, o := io.Pipe()
	b := bufio.NewReader(i)
	go func() {
		matchables := Mix(common, Cardinality-CommonCardinality, HashLen)
		for matchable := range matchables {
			out := append(matchable, "\n"...)
			if _, err := o.Write(out); err != nil {
				return
			}
		}
	}()
	return b
}

func TestGenerate(t *testing.T) {
	// generate common data
	common := Common(CommonCardinality, HashLen)
	r := initDataSource(common)

	// read N matchables from r
	// and write them to stage1
	for i := 0; i < Cardinality; i++ {
		line, _, err := r.ReadLine()
		if err != nil {
			t.Fatalf("not error expected, got error %v", err)
		}
		if !strings.HasPrefix(string(line), Prefix) {
			t.Fatalf("expected prefix %s, got %s", Prefix, string(line[:1]))
		}
	}
}
