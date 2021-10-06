// black box testing of all PSIs
package psi_test

type test_size struct {
	scenario                          string
	commonLen, senderLen, receiverLen int
}

// test scenarios
// the common part will be subtracted from the sender &
// the receiver len, so for instance
//
//  100 common, 100 sender will result in the sender len being 100 and only
//  composed of the common part
//
var test_sizes = []test_size{
	{"sender100receiver200", 100, 100, 200},
	{"emptySenderSize", 0, 0, 1000},
	{"emptyReceiverSize", 0, 1000, 0},
	{"sameSize", 100, 100, 100},
	{"smallSize", 100, 10000, 1000},
	{"mediumSize", 1000, 100000, 10000},
	{"bigSize", 10000, 100000, 100000},
}
