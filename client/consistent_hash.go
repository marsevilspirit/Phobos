package client

import (
	"fmt"
	"hash/fnv"
)

func Hash(key uint64, buckets int32) int32 {
	if buckets <= 0 {
		buckets = 1
	}

	var b, j int64
	for j < int64(buckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(int64(1)<<31)/float64(key>>33) + 1))
	}

	return int32(b)
}

func HashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func JumpConsistentHash(len int, options ...any) int {
	var keyString string
	for _, opt := range options {
		keyString = keyString + "/" + toString(opt)
	}
	key := HashString(keyString)
	return int(Hash(key, int32(len)))
}

func toString(obj any) string {
	return fmt.Sprintf("%v", obj)
}

// HashServiceAndArgs define a hash function
type HashServiceAndArgs func(len int, options ...any) int

// ConsistentFunction define a hash function
// Return service address, like "tcp@127.0.0.1:8970"
type ConsistentAddrStrFunction func(options ...any) string
