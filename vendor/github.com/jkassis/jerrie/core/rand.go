package core

import (
	"hash/maphash"
	"math"
	"math/rand"
	"time"
)

// RandInt returns a random integer between min (inclusive) and max (exclusive)
func RandInt(min, max int) int {
	return rand.Intn(max-min) + min
}

// RandBackoffRetryDelay returns a random delay in an exponentially growing range
// skip can be used to to skip ranges. eg. skip == 2 makes the first range 4..8
// mult can be used to increase the exponential of the random backoff.
// e.g. mult == 2 goes from 1..2 to 4..8 to 16..32 to
// 1..2 seconds
// 2..4 seconds
// 4..8 seconds
// 8..16 seconds
// 16..32 seconds
// 32..64 seconds
// 64..128 seconds
// 128..256 seconds
// a good patern for networked services is...
// RandomBackoffRetryDelay(i, 1, 2) with i from [0..3]
func RandBackoffRetryDelay(attempt, skip, mult int) time.Duration {
	min := int64(1000 * math.Pow(2, float64(mult*attempt+skip)))
	max := int64(1000 * math.Pow(2, float64(mult*attempt+skip+1)))
	return time.Duration(rand.Int63n(max-min)+min) * time.Millisecond
}

// Rand64Fast returns a pseudo-random uint64. It can be used concurrently and is lock-free.
// Effectively, it calls runtime.fastrand.
func Rand64Fast() uint64 {
	return new(maphash.Hash).Sum64()
}

// NewRand64Fast returns a properly seeded *rand.Rand. It has *slightly* higher overhead than
// Rand64 (as it has to allocate), but the resulting PRNG can be re-used to offset that cost.
// Use this if you can't just mask off bits from a uint64 (e.g. if you need to use Intn() with
// something that is not a power of 2).
func NewRand64Fast() *rand.Rand {
	return rand.New(rand.NewSource(int64(Rand64Fast())))
}
