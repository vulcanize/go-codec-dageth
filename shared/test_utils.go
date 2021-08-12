package shared

import (
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// RandomHash returns a random hash
func RandomHash() common.Hash {
	rand.Seed(time.Now().UnixNano())
	hash := make([]byte, 32)
	rand.Read(hash)
	return common.BytesToHash(hash)
}

// RandomAddr returns a random address
func RandomAddr() common.Address {
	rand.Seed(time.Now().UnixNano())
	addr := make([]byte, 20)
	rand.Read(addr)
	return common.BytesToAddress(addr)
}

// RandomBytes returns a random byte slice of the provided length
func RandomBytes(len int) []byte {
	rand.Seed(time.Now().UnixNano())
	by := make([]byte, len)
	rand.Read(by)
	return by
}
