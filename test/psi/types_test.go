// black box testing of all PSIs
package psi_test

import "github.com/optable/match/test/emails"

type test_size struct {
	scenario                                   string
	commonLen, senderLen, receiverLen, hashLen int
}

// test scenarios
// the common part will be subtracted from the sender &
// the receiver len, so for instance
//
//  100 common, 100 sender will result in the sender len being 100 and only
//  composed of the common part
//
var test_sizes = []test_size{
	{"sender100receiver200", 100, 100, 200, emails.HashLen},
	{"emptySenderSize", 0, 0, 1000, emails.HashLen},
	{"emptyReceiverSize", 0, 1000, 0, emails.HashLen},
	{"sameSize", 100, 100, 100, emails.HashLen},
	{"smallSize", 100, 10000, 1000, emails.HashLen},
	{"mediumSize", 1000, 100000, 10000, emails.HashLen},
	{"bigSize", 10000, 100000, 100000, emails.HashLen},
}

var hashLenSizes = []int{4, 8, 16, 32, 64}
