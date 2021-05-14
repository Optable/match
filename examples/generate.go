package main

import (
	"log"
	"os"
	"sync"

	"github.com/optable/match/test/emails"
)

const (
	senderCardinality   = 1000
	senderFileName      = "sender-ids.txt"
	receiverCardinality = 10000
	receiverFileName    = "receiver-ids.txt"
	commonCardinality   = senderCardinality / 10
)

func main() {
	var ws sync.WaitGroup
	// make the common part
	common := emails.Common(commonCardinality)
	// do advertisers & publishers in parallel
	ws.Add(2)
	go output(senderFileName, common, senderCardinality-commonCardinality, &ws)
	go output(receiverFileName, common, receiverCardinality-commonCardinality, &ws)
	ws.Wait()
}

func output(filename string, common []byte, n int, ws *sync.WaitGroup) {
	defer ws.Done()
	if f, err := os.Create(filename); err == nil {
		defer f.Close()
		out := emails.Mix(common, n)
		// exhaust out
		for matchable := range out {
			// and write it
			if _, err := f.Write(matchable); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		log.Fatal(err)
	}
}
