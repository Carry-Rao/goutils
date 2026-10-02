package bloom

import "sync/atomic"

const bloomSize = 1024
const bloomWords = bloomSize / 64

type BloomFilter struct {
	bits [bloomWords]atomic.Uint64
}

func (bf *BloomFilter) Add(h1, h2 uint64) {
	for i := 0; i < 3; i++ {
		idx := (h1 + uint64(i)*h2) % bloomSize
		word := idx / 64
		bit := idx % 64
		bf.bits[word].Or(1 << bit)
	}
}

func (bf *BloomFilter) Contains(h1, h2 uint64) bool {
	for i := 0; i < 3; i++ {
		idx := (h1 + uint64(i)*h2) % bloomSize
		word := idx / 64
		bit := idx % 64
		if bf.bits[word].Load()&(1<<bit) == 0 {
			return false
		}
	}
	return true
}
