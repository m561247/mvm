package vm

import (
	"math"
	"reflect"
)

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type float interface{ ~float32 | ~float64 }

func add[T integer](a, b uint64) uint64 { return uint64(T(a) + T(b)) }
func sub[T integer](a, b uint64) uint64 { return uint64(T(a) - T(b)) }
func mul[T integer](a, b uint64) uint64 { return uint64(T(a) * T(b)) }
func div[T integer](a, b uint64) uint64 { return uint64(T(a) / T(b)) }
func rem[T integer](a, b uint64) uint64 { return uint64(T(a) % T(b)) }
func neg[T integer](a uint64) uint64    { return uint64(-T(a)) }

func addf[T float](a, b uint64) uint64 {
	return math.Float64bits(float64(T(math.Float64frombits(a)) + T(math.Float64frombits(b))))
}

func subf[T float](a, b uint64) uint64 {
	return math.Float64bits(float64(T(math.Float64frombits(a)) - T(math.Float64frombits(b))))
}

func mulf[T float](a, b uint64) uint64 {
	return math.Float64bits(float64(T(math.Float64frombits(a)) * T(math.Float64frombits(b))))
}

func divf[T float](a, b uint64) uint64 {
	return math.Float64bits(float64(T(math.Float64frombits(a)) / T(math.Float64frombits(b))))
}

func negf[T float](a uint64) uint64 {
	return math.Float64bits(float64(-T(math.Float64frombits(a))))
}

// getf32 extracts a float32 from a Value's uint64 storage (float64-bits encoding).
func getf32(n uint64) float32 { return float32(math.Float64frombits(n)) }

// putf32 stores a float32 into a Value's uint64 storage (float64-bits encoding).
func putf32(f float32) uint64 { return math.Float64bits(float64(f)) }

// truncToKind narrows n to k's value width, sign-extending for signed kinds.
// Restores the invariant that Value.num holds the typed value in uint64 form
// after ops (BitShl, BitComp) that operate on uint64 without respecting width.
func truncToKind(n uint64, k reflect.Kind) uint64 {
	switch k {
	case reflect.Int8:
		return uint64(int64(int8(n)))
	case reflect.Int16:
		return uint64(int64(int16(n)))
	case reflect.Int32:
		return uint64(int64(int32(n)))
	case reflect.Uint8:
		return uint64(uint8(n))
	case reflect.Uint16:
		return uint64(uint16(n))
	case reflect.Uint32:
		return uint64(uint32(n))
	}
	return n
}
