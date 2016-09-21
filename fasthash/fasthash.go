package fasthash

import (
	"crypto/md5"
	"encoding/binary"
	"errors"
	"github.com/dgryski/go-jump"
	"sort"
)

var (
	// ErrNoMembers occurs when trying to hash with no members
	ErrNoMembers = errors.New("no members available")
)

// FastHash holds the internal workings for the hashing
type FastHash struct {
	keys []string
}

// Creates a new byteJumper pointer
func New(keys []string) *FastHash {
	fast := new(FastHash)
	sort.Strings(keys)
	fast.keys = keys

	return fast
}

// Get finds the member for a given key
func (fast *FastHash) Get(key []byte) (string, error) {
	keyCount := len(fast.keys)

	if 0 == keyCount {
		return "", ErrNoMembers
	}

	hash := md5.Sum(key)
	index := jump.Hash(binary.BigEndian.Uint64(hash[8:16]), keyCount)

	return fast.keys[index], nil
}
