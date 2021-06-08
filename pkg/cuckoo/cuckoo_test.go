package cuckoo

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math"
	"testing"
)

var n = uint64(32)
var Identifiers = [32][]byte{
	[]byte("e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e"),
	[]byte("e:73244e1b8c426ed93d315034d9332d5326c6b0cd72cc49093c25106f0dd081c5"),
	[]byte("e:e14efb6bb979cd467767d6d90bd9b4f1d901396eabaa90384e747a00d820776d"),
	[]byte("e:402b44cf09b3004c23257d4b9dc39b0a46966865529393f4533048b5e156ad90"),
	[]byte("e:d03ef68830b089a25592cca16fe3eae40a76ddacdd62719c3ff5eb780e4e490f"),
	[]byte("e:69951783d7e4ae1df6754a517d45e40a29940d91c748ffb53e866adef10a78a1"),
	[]byte("e:46cf5044da24ef9f7b368e9130a9f67b2f4ea22d1a9d403898bb59b5ee391385"),
	[]byte("e:2f3a9eb79657279fed578fb9c038fa8bd5eb40ec2c8b23a24a2bde64d2571138"),
	[]byte("e:a74401c671bc8bd23739f2c2fcc55166532500aa3a63b572d68e7059345fffed"),
	[]byte("e:4a820b8e791a43f265e6a32d330026e934dd29e38095a2b25f238c39b8bf434d"),
	[]byte("e:f9d2a0735baeb9b35c657309d7187b00e10965f70541cebbfb5499a36be0e283"),
	[]byte("e:d6c3f32ee1324b0a1fd3b8f2a338cb49b39e240583b43eabb16182d291e7aa39"),
	[]byte("e:7ab89627155ecc540d9237eb2963d36f08e57bf9f5e12fab40317d7843efd862"),
	[]byte("e:fece4ff2fae77d65e01ef57fe39c54cc6cf0eab1547c3feee961a6a30f183431"),
	[]byte("e:74de73ee6b30ac0a8d93a7f871a12e518438496954c5052e0082591188ccaff0"),
	[]byte("e:3d49a32e9cce74193aa9fbcb678e20a1efb24f5a2dc296d9fdf4ac445abc1533"),
	[]byte("e:6e30b02a1caae75b0b3ac5e4b63f0f6d3ce8996edcef3c5788e67dc5f8f7c014"),
	[]byte("e:c6005e8fd6de80fc0503700ad64b452a0d2edacef538b17fa8792312f2f8f1d6"),
	[]byte("e:4ed837fad590878b62a7a2118704f8e48a3aa3831ac49d47063a3ca9616fa381"),
	[]byte("e:47618a0fa83a71f3bfa8c084c579c4ec6b82d2fafc913db2aeed102582f6ceaa"),
	[]byte("e:708945bb65132978dbc96aed8fe41083d1637ba8dec9803cdea2a9b7e1da7e37"),
	[]byte("e:bfbb0c6984d691d15bcc620e73223de1cc90cdd16831eefe68c9ac736f849094"),
	[]byte("e:4dff07a5d432b82fdc5112ad9a5110fcb959223723851f8db23bca5077b51b7d"),
	[]byte("e:3e0fd04d1c279bd43a7d1f33bc6cae5350f44bb6cc569cd721e69e8e417bbb9c"),
	[]byte("e:9f9fcfc832dbc978d39eaef9d517be61068778e02e784db8d3cb37d8bffc6523"),
	[]byte("e:915c9539ddbd4291f6acfa33d0107cdff80f97f4572751f753450d18141f8359"),
	[]byte("e:9cf8e9ee2ee9835744d698351851ede35189d73dffed5e250c5662bfa443ff51"),
	[]byte("e:6d54013abd77c0eb5de7a759d413d650e52430141b86abb00a93b7c6d756e128"),
	[]byte("e:021502364ce760272a61416f59ee6ee85cff14542483df4b3baa6517a2d94b92"),
	[]byte("e:4fcf5589928226c510303586952213963ebcfcc2a1cd84bdf4f6062714ebbe99"),
	[]byte("e:1ae75243caec797321f30a8a566b62461163b340bf0982f1a0199c8898c7f799"),
	[]byte("e:693f16e5f2b814c592a6ab7478da155907362b42634d99a78b15fd7b1886faa7"),
}

func makeSeeds() [Nhash][]byte {
	var seeds [Nhash][]byte

	for i, _ := range seeds {
		seeds[i] = make([]byte, 32)
		if _, err := rand.Read(seeds[i]); err != nil {
			seeds[i] = nil
		}
	}

	return seeds
}

func TestStashSize(t *testing.T) {
	// Table driven test
	stashSizeTests := []struct {
		n    uint64 //input size
		want uint8  // stash size
	}{
		{uint64(0), uint8(0)},
		{uint64(math.Pow(2, 8) - 1), uint8(12)},
		{uint64(math.Pow(2, 12) - 1), uint8(6)},
		{uint64(math.Pow(2, 16) - 1), uint8(4)},
		{uint64(math.Pow(2, 20) - 1), uint8(3)},
		{uint64(math.Pow(2, 24)), uint8(2)},
		{uint64(math.Pow(2, 25)), uint8(0)},
	}

	for _, tt := range stashSizeTests {
		got := findStashSize(tt.n)
		if got != tt.want {
			t.Errorf("findStashSize(%d): want: %d, got: %d", tt.n, tt.want, got)
		}
	}
}

func TestNewCuckoo(t *testing.T) {
	seeds := makeSeeds()

	cuckooTests := []struct {
		size  uint64
		bSize uint64 //bucketSize
	}{
		{uint64(0), uint64(0)},
		{uint64(math.Pow(2, 4)), uint64(1.2 * math.Pow(2, 4))},
		{uint64(math.Pow(2, 8)), uint64(1.2 * math.Pow(2, 8))},
		{uint64(math.Pow(2, 16)), uint64(1.2 * math.Pow(2, 16))},
	}

	for _, tt := range cuckooTests {
		c := NewCuckoo(tt.size, seeds)
		if c.bucketSize != tt.bSize {
			t.Errorf("cuckoo bucketsize: want: %d, got: %d", tt.bSize, c.bucketSize)
		}
	}
}

func TestInsertAndGetHashIdx(t *testing.T) {
	seeds := makeSeeds()

	cuckoo := NewCuckoo(n, seeds)

	//test Insert
	for _, item := range Identifiers {
		err := cuckoo.Insert(item)
		if err != nil {
			t.Errorf("Cuckoo insert failed: %w", err)
		}
	}

	//test GetHashIdx
	for _, item := range Identifiers {
		idx, found := cuckoo.GetHashIdx(item)
		if !found {
			t.Errorf("Cuckoo GetHashIdx, item: %s not inserted.", string(item[:]))
		}

		if idx != StashHidx {
			bIdx := cuckoo.bucketIndex(cuckoo.hash(item)[idx])
			if !bytes.Equal(cuckoo.buckets[bIdx], item) {
				t.Errorf("Cuckoo GetHashIdx, hashIdx not correct for item: %s", string(item[:]))
			}
		} else {
			found = false
			for _, v := range cuckoo.stash {
				if bytes.Equal(v, item) {
					found = true
				}
			}
			if !found {
				t.Errorf("Cuckoo GetHashIdx, hashIdx is StashHidx but not found in stash for item: %s", string(item[:]))
			}
		}
	}

	//debug
	//printBucket(cuckoo)
	//printStash(cuckoo)
}

// Benchmark insertion
func BenchmarkCuckooInsert(b *testing.B) {
	seeds := makeSeeds()
	cuckoo := NewCuckoo(n, seeds)

	for i := 0; i < b.N; i++ {
		cuckoo.Insert(Identifiers[i%len(Identifiers)])
	}
}

// Benchmark find hash index
func BenchmarkCuckooGetHashIdx(b *testing.B) {
	seeds := makeSeeds()
	cuckoo := NewCuckoo(n, seeds)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cuckoo.GetHashIdx(Identifiers[i%len(Identifiers)])
	}
}

func printBucket(c *Cuckoo) {
	for k, v := range c.buckets {
		fmt.Printf("bIdx: %d, item: %s\n", k, string(v[:]))
	}
}

func printStash(c *Cuckoo) {
	for _, s := range c.stash {
		fmt.Printf("item: %s", string(s[:]))
	}
}
