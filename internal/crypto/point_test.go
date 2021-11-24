package crypto

import (
	"fmt"
	"math/big"
	"testing"
)

type addTest struct {
	xLeft, yLeft   string
	xRight, yRight string
	xOut, yOut     string
}

var addTests = []addTest{
	{
		"48439561293906451759052585252797914202762949526041747995844080717082404635286", // base point X
		"36134250956749795798585127919587881956611106672985015071877198253568414405109", // base point Y
		"48439561293906451759052585252797914202762949526041747995844080717082404635286",
		"36134250956749795798585127919587881956611106672985015071877198253568414405109",
		"56515219790691171413109057904011688695424810155802929973526481321309856242040", // 2x
		"3377031843712258259223711451491452598088675519751548567112458094635497583569",  // 2y
	},
	{
		"48439561293906451759052585252797914202762949526041747995844080717082404635286",  // base point X
		"36134250956749795798585127919587881956611106672985015071877198253568414405109",  // base point Y
		"102369864249653057322725350723741461599905180004905897298779971437827381725266", // 4x
		"101744491111635190512325668403432589740384530506764148840112137220732283181254", // 4y
		"36794669340896883012101473439538929759152396476648692591795318194054580155373",  // 5x
		"101659946828913883886577915207667153874746613498030835602133042203824767462820", // 5y
	},
}

func TestAdd(t *testing.T) {
	for i, e := range addTests {
		xL, _ := new(big.Int).SetString(e.xLeft, 10)
		yL, _ := new(big.Int).SetString(e.yLeft, 10)
		xR, _ := new(big.Int).SetString(e.xRight, 10)
		yR, _ := new(big.Int).SetString(e.yRight, 10)
		pointL := &Point{x: xL, y: yL}
		pointR := &Point{x: xR, y: yR}
		expectedX, _ := new(big.Int).SetString(e.xOut, 10)
		expectedY, _ := new(big.Int).SetString(e.yOut, 10)
		sum1 := pointL.Add(pointR)
		if !sum1.Equal(&Point{x: expectedX, y: expectedY}) {
			t.Errorf("#%d: got (%s, %s), want (%s, %s)", i, sum1.x.String(), sum1.y.String(), expectedX.String(), expectedY.String())
		}

		sum2 := pointL.Add(pointR)
		if !sum2.Equal(&Point{x: expectedX, y: expectedY}) {
			t.Errorf("#%d: got (%s, %s), want (%s, %s)", i, sum2.x.String(), sum2.y.String(), expectedX.String(), expectedY.String())
		}
	}
}

type scalarMultTest struct {
	k          string
	xIn, yIn   string
	xOut, yOut string
}

var scalarMultTests = []scalarMultTest{
	{
		"2a265f8bcbdcaf94d58519141e578124cb40d64a501fba9c11847b28965bc737",
		"023819813ac969847059028ea88a1f30dfbcde03fc791d3a252c6b41211882ea",
		"f93e4ae433cc12cf2a43fc0ef26400c0e125508224cdb649380f25479148a4ad",
		"4d4de80f1534850d261075997e3049321a0864082d24a917863366c0724f5ae3",
		"a22d2b7f7818a3563e0f7a76c9bf0921ac55e06e2e4d11795b233824b1db8cc0",
	},
	{
		"313f72ff9fe811bf573176231b286a3bdb6f1b14e05c40146590727a71c3bccd",
		"cc11887b2d66cbae8f4d306627192522932146b42f01d3c6f92bd5c8ba739b06",
		"a2f08a029cd06b46183085bae9248b0ed15b70280c7ef13a457f5af382426031",
		"831c3f6b5f762d2f461901577af41354ac5f228c2591f84f8a6e51e2e3f17991",
		"93f90934cd0ef2c698cc471c60a93524e87ab31ca2412252337f364513e43684",
	},
}

func TestScalarMult(t *testing.T) {
	for i, e := range scalarMultTests {
		x, _ := new(big.Int).SetString(e.xIn, 16)
		y, _ := new(big.Int).SetString(e.yIn, 16)
		k, _ := new(big.Int).SetString(e.k, 16)
		point := &Point{x: x, y: y}
		expectedX, _ := new(big.Int).SetString(e.xOut, 16)
		expectedY, _ := new(big.Int).SetString(e.yOut, 16)

		kPoint := point.ScalarMult(k.Bytes())
		if !kPoint.Equal(&Point{x: expectedX, y: expectedY}) {
			t.Errorf("#%d: got (%x, %x), want (%x, %x)", i, kPoint.x, kPoint.y, expectedX, expectedY)
		}
	}
}

func TestSub(t *testing.T) {
	for i, e := range addTests {
		expectedX, _ := new(big.Int).SetString(e.xLeft, 10)
		expectedY, _ := new(big.Int).SetString(e.yLeft, 10)
		x, _ := new(big.Int).SetString(e.xRight, 10)
		y, _ := new(big.Int).SetString(e.yRight, 10)
		xSum, _ := new(big.Int).SetString(e.xOut, 10)
		ySum, _ := new(big.Int).SetString(e.yOut, 10)
		point := &Point{x: x, y: y}
		sum := &Point{x: xSum, y: ySum}

		diff := sum.Sub(point)
		if !diff.Equal(&Point{x: expectedX, y: expectedY}) {
			t.Errorf("#%d: got (%s, %s), want (%s, %s)", i, diff.x.String(), diff.y.String(), expectedX.String(), expectedY.String())
		}
	}
}

type keyMarshalTest struct {
	x, y    string
	key     string
	marshal string
}

var keyMarshalTests = []keyMarshalTest{
	{
		"48439561293906451759052585252797914202762949526041747995844080717082404635286", // base point X
		"36134250956749795798585127919587881956611106672985015071877198253568414405109", // base point Y
		"f7f366d0495aeb2267bc93104770f2ccc28929f610575a0ccf43b5a8fa53febb",
		"036b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296",
	},
	{
		"102369864249653057322725350723741461599905180004905897298779971437827381725266", // 4x
		"101744491111635190512325668403432589740384530506764148840112137220732283181254", // 4y
		"4354ebf0fc87975a838e3d8af6f5bb074e5cba896843ec4f0ddf670f343c0e8a",
		"02e2534a3532d08fbba02dde659ee62bd0031fe2db785596ef509302446b030852",
	},
}

func TestDeriveKeyPoint(t *testing.T) {
	for i, e := range keyMarshalTests {
		x, _ := new(big.Int).SetString(e.x, 10)
		y, _ := new(big.Int).SetString(e.y, 10)
		point := &Point{x: x, y: y}
		key := point.DeriveKeyFromECPoint()

		if fmt.Sprintf("%x", key) != e.key {
			t.Errorf("#%d: got %x, want %v", i, key, e.key)
		}
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	for i, e := range keyMarshalTests {
		x, _ := new(big.Int).SetString(e.x, 10)
		y, _ := new(big.Int).SetString(e.y, 10)
		point := &Point{x: x, y: y}

		marshaled := point.Marshal()
		if fmt.Sprintf("%x", marshaled) != e.marshal {
			t.Errorf("#%d: got %x, want %v", i, marshaled, e.marshal)
		}

		unmarshalPoint := NewPoint()
		unmarshalPoint.Unmarshal(marshaled)
		if !point.Equal(unmarshalPoint) {
			t.Errorf("#%d: got (%x, %x), want (%x, %x)", i, unmarshalPoint.x, unmarshalPoint.y, point.x, point.y)
		}
	}
}

func BenchmarkDeriveKey(b *testing.B) {
	p := &Point{x: big.NewInt(1), y: big.NewInt(2)}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.DeriveKeyFromECPoint()
	}
}

func BenchmarkSub(b *testing.B) {
	p := &Point{x: big.NewInt(1), y: big.NewInt(2)}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Sub(p)
	}
}
