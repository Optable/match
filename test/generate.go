package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/optable/match/test/emails"
)

// from fake_hashes.sh:
// 	Usage: fake_hashes.sh n_hashes length prefix
//
// generate n_hashes of length length and prefix them with prefix
//
// example:
//  e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
//  e:59245d7c68b28404e068b15cba430082549b845ab412c4c3b31fb8632fd794e1
//  e:8d4acbaaec5a4b00465fa6db04deeb7de8722ef2893a3c22096fafe060686c38
//  e:63bc6f65441e7650fd5d3add5f16029bd44abb6af16eeeb532ddfd85865706b5
//  e:d427b4d8dfcfcd4e972d3f836da9457ac0e2dce26f46d9efd264d7af2440b892
//  e:e33da25066c2d7d959de91844f593ac6a8991829f4fda71a2aefb9445745cca1
//  e:6a534d2073cf17f6f42784934dd3b6f3776a9ebd28a2a1594c0f5e0ff3d58002
//  e:fc7f9988396ef0144486c4c7e108f8d3833024d6931e4dc554fc9b1f581f5260
//
// hashes are random blobs of length length expressed in hex and prefixed with a string

const (
	usage = `%s cardinality_of_sender cardinality_of_receiver number_in_common (min(a,p)/10) sender_output_file (%s) receiver_output_file (%s)

 the default size of the common portion is min(cardinality_of_sender, cardinality_of_receiver) / 10

example:
 %s 100000 1000000
`
	defaultSenderCardinality   = 100000
	defaultReceiverCardinality = 1000000
	defaultSenderOutput        = "sender.txt"
	defaultReceiverOutput      = "receiver.txt"
)

type config struct {
	senderCardinality   int
	receiverCardinality int
	common              int
	senderOutput        string
	receiverOutput      string
}

func formatUsage() string {
	name := os.Args[0]
	return fmt.Sprintf(usage, name, name, defaultSenderOutput, defaultReceiverOutput)
}

// global conf
var conf config

func formatArgs() string {
	return fmt.Sprintf("generating %d for the sender and %d for the receiver with %d in common to %s and %s",
		conf.senderCardinality, conf.receiverCardinality, conf.common, conf.senderOutput, conf.receiverOutput)
}

// min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	// we have default values for everything
	// this would probably better off be in something like envconfig
	if len(os.Args) > 1 {
		if snd, err := strconv.Atoi(os.Args[1]); err == nil {
			conf.senderCardinality = snd
		} else {
			log.Fatal(err)
		}
	} else {
		conf.senderCardinality = defaultSenderCardinality
	}
	if len(os.Args) > 2 {
		if rcv, err := strconv.Atoi(os.Args[2]); err == nil {
			conf.receiverCardinality = rcv
		} else {
			log.Fatal(err)
		}
	} else {
		conf.receiverCardinality = defaultReceiverCardinality
	}
	// common
	if len(os.Args) > 3 {
		if common, err := strconv.Atoi(os.Args[3]); err == nil {
			conf.common = common
		} else {
			log.Fatal(err)
		}
	} else {
		conf.common = min(conf.senderCardinality, conf.receiverCardinality) / 10
	}
	// senderOutput
	if len(os.Args) > 4 {
		conf.senderOutput = os.Args[4]
	} else {
		conf.senderOutput = defaultSenderOutput
	}
	// receiverOutput
	if len(os.Args) > 5 {
		conf.receiverOutput = os.Args[5]
	} else {
		conf.receiverOutput = defaultReceiverOutput
	}
}

func main() {
	var ws sync.WaitGroup
	println(formatUsage())
	// make the common part
	common := emails.Common(conf.common)
	println(formatArgs())
	// do advertisers & publishers in parallel
	ws.Add(2)
	go output(conf.senderOutput, common, conf.senderCardinality-conf.common, &ws)
	go output(conf.receiverOutput, common, conf.receiverCardinality-conf.common, &ws)
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
