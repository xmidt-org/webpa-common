package aws

import (
	"crypto"
	"crypto/rsa"
	"crypto/subtle"
	"errors"
//	"io"
	"math/big"
	"fmt"
)
/*
const (
	ErrVerification = "crypto/rsa: verification error"
)
*/

func leftPad(input []byte, size int) (out []byte) {
		n := len(input)
		if n > size {
			n = size
		}
		out = make([]byte, size)
		copy(out[len(out)-n:], input)
		return
	}

func encrypt(c *big.Int, pub *rsa.PublicKey, m *big.Int) *big.Int {
		e := big.NewInt(int64(pub.E))
		c.Exp(m, e, pub.N)
		return c
	}

var hashPrefixes = map[crypto.Hash][]byte{
		crypto.MD5:       {0x30, 0x20, 0x30, 0x0c, 0x06, 0x08, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x02, 0x05, 0x05, 0x00, 0x04, 0x10},
		crypto.SHA1:      {0x30, 0x21, 0x30, 0x09, 0x06, 0x05, 0x2b, 0x0e, 0x03, 0x02, 0x1a, 0x05, 0x00, 0x04, 0x14},
		crypto.SHA224:    {0x30, 0x2d, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x04, 0x05, 0x00, 0x04, 0x1c},
		crypto.SHA256:    {0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05, 0x00, 0x04, 0x20},
		crypto.SHA384:    {0x30, 0x41, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02, 0x05, 0x00, 0x04, 0x30},
		crypto.SHA512:    {0x30, 0x51, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03, 0x05, 0x00, 0x04, 0x40},
		crypto.MD5SHA1:   {}, // A special TLS case which doesn't use an ASN1 prefix.
		crypto.RIPEMD160: {0x30, 0x20, 0x30, 0x08, 0x06, 0x06, 0x28, 0xcf, 0x06, 0x03, 0x00, 0x31, 0x04, 0x14},
	}

func hashInfo(hash crypto.Hash, inLen int) (hashLen int, prefix []byte, err error) {
		// Special case: crypto.Hash(0) is used to indicate that the data is
		// signed directly.
		if hash == 0 {
			return inLen, nil, nil
		}
	
		hashLen = hash.Size()
		if inLen != hashLen {
			return 0, nil, errors.New("crypto/rsa: input must be hashed message")
		}
		prefix, ok := hashPrefixes[hash]
		if !ok {
			return 0, nil, errors.New("crypto/rsa: unsupported hash function")
		}
		return
	}

func verify(pub *rsa.PublicKey, hash crypto.Hash, hashed []byte, sig []byte) error {
		hashLen, prefix, err := hashInfo(hash, len(hashed))
		if err != nil {
			return err
		}
	
		tLen := len(prefix) + hashLen
		k := (pub.N.BitLen() + 7) / 8
		if k < tLen+11 {
			fmt.Println("error with size")
			return rsa.ErrVerification
		}
	
		c := new(big.Int).SetBytes(sig)
		m := encrypt(new(big.Int), pub, c)
		em := leftPad(m.Bytes(), k)
		// EM = 0x00 || 0x01 || PS || 0x00 || T
	
		ok := subtle.ConstantTimeByteEq(em[0], 0)
		fmt.Printf("1:%v\n", ok)
		ok &= subtle.ConstantTimeByteEq(em[1], 1)
		fmt.Printf("2:%v\n", ok)
		ok &= constantTimeCompare(em[k-hashLen:k], hashed)
		fmt.Printf("3:%v\n", ok)
		ok &= subtle.ConstantTimeCompare(em[k-tLen:k-hashLen], prefix)
		fmt.Printf("4:%v\n", ok)
		ok &= subtle.ConstantTimeByteEq(em[k-tLen-1], 0)
		fmt.Printf("5:%v\n", ok)
	
		for i := 2; i < k-tLen-1; i++ {
			ok &= subtle.ConstantTimeByteEq(em[i], 0xff)
			fmt.Printf("6:%v\n", ok)
		}
	
		if ok != 1 {
			fmt.Println("dive deeper, verification error")
			return rsa.ErrVerification
		}
	
		return nil
	}

func constantTimeByteEq(x, y uint8) int {
	z := ^(x ^ y)
	fmt.Printf("1 z: %v\n", z)
	z &= z >> 4
	fmt.Printf("2 z: %v\n", z)
	z &= z >> 2
	fmt.Printf("3 z: %v\n", z)
	z &= z >> 1
	fmt.Printf("4 z: %v\n", z)

	v := int(z)
	
	fmt.Printf("value of v: %v\n", v)
	
	return v
}
	
func constantTimeCompare(x, y []byte) int {
	if len(x) != len(y) {
		fmt.Println("len(x) != len(y)")
		return 0
	}

	var v byte

	for i := 0; i < len(x); i++ {
		v |= x[i] ^ y[i]
	}

	return constantTimeByteEq(v, 0)
}