package stdlib

import (
	"math"
	"math/bits"
)

// trigReduce, mPi4, and reduceThreshold are ported verbatim from
// $GOROOT/src/math/trig_reduce.go (BSD-licensed, The Go Authors) so that the
// math export_test.go symbols TrigReduce and ReduceThreshold exist for
// `mvm test math`'s external suite (math is a native bridge, so its
// test-only internal symbols are otherwise absent). Keep in sync if the
// upstream algorithm changes.

// reduceThreshold is the largest x where Pi/4 reduction in 3 float64 parts is
// still accurate; above it Payne-Hanek reduction is required.
const reduceThreshold = 1 << 29

// trigReduce implements Payne-Hanek range reduction by Pi/4 for x > 0. It
// returns the integer part mod 8 (j) and the fractional part (z) of x / (Pi/4).
func trigReduce(x float64) (j uint64, z float64) {
	const (
		mask  = 0x7FF
		shift = 64 - 11 - 1
		bias  = 1023
		pi4   = math.Pi / 4
	)
	if x < pi4 {
		return 0, x
	}
	// Extract the integer and exponent such that x = ix * 2**exp.
	ix := math.Float64bits(x)
	exp := int(ix>>shift&mask) - bias - shift
	ix &^= mask << shift
	ix |= 1 << shift
	// Use the exponent to extract the 3 appropriate uint64 digits from mPi4,
	// such that the product's leading digit has exponent -61.
	digit, bitshift := uint(exp+61)/64, uint(exp+61)%64
	z0 := (mPi4[digit] << bitshift) | (mPi4[digit+1] >> (64 - bitshift))
	z1 := (mPi4[digit+1] << bitshift) | (mPi4[digit+2] >> (64 - bitshift))
	z2 := (mPi4[digit+2] << bitshift) | (mPi4[digit+3] >> (64 - bitshift))
	// Multiply the mantissa by the digits and extract the upper two digits.
	z2hi, _ := bits.Mul64(z2, ix)
	z1hi, z1lo := bits.Mul64(z1, ix)
	z0lo := z0 * ix
	lo, c := bits.Add64(z1lo, z2hi, 0)
	hi, _ := bits.Add64(z0lo, z1hi, c)
	// The top 3 bits are j.
	j = hi >> 61
	// Extract the fraction and find its magnitude.
	hi = hi<<3 | lo>>61
	lz := uint(bits.LeadingZeros64(hi))
	e := uint64(bias - (lz + 1))
	hi = (hi << (lz + 1)) | (lo >> (64 - (lz + 1)))
	hi >>= 64 - shift
	// Include the exponent and convert to a float.
	hi |= e << shift
	z = math.Float64frombits(hi)
	// Map zeros to origin.
	if j&1 == 1 {
		j++
		j &= 7
		z--
	}
	return j, z * pi4
}

// mPi4 is the binary digits of 4/pi as a uint64 array (4/pi = Sum
// mPi4[i]*2^(-64*i)); 19 digits plus the leading one bit give 1217 bits.
var mPi4 = [...]uint64{
	0x0000000000000001,
	0x45f306dc9c882a53,
	0xf84eafa3ea69bb81,
	0xb6c52b3278872083,
	0xfca2c757bd778ac3,
	0x6e48dc74849ba5c0,
	0x0c925dd413a32439,
	0xfc3bd63962534e7d,
	0xd1046bea5d768909,
	0xd338e04d68befc82,
	0x7323ac7306a673e9,
	0x3908bf177bf25076,
	0x3ff12fffbc0b301f,
	0xde5e2316b414da3e,
	0xda6cfd9e4f96136e,
	0x9e8c7ecd3cbfd45a,
	0xea4f758fd7cbe2f6,
	0x7a0e73ef14a525d4,
	0xd7f6bf623f1aba10,
	0xac06608df8f6d757,
}
