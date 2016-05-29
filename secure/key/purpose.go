package key

import (
	"bytes"
	"fmt"
	"github.com/SermoDigital/jose/crypto"
)

// Purpose is an enumerated type describing the reason a given
// key is being used
type Purpose int

const (
	PurposeSign Purpose = Purpose(iota)
	PurposeVerify
	PurposeEncrypt
	PurposeDecrypt
)

var (
	purposeMarshal = map[Purpose]string{
		PurposeSign:    "sign",
		PurposeVerify:  "verify",
		PurposeEncrypt: "encrypt",
		PurposeDecrypt: "decrypt",
	}

	purposeUnmarshal = map[string]Purpose{
		"sign":    PurposeSign,
		"verify":  PurposeVerify,
		"encrypt": PurposeEncrypt,
		"decrypt": PurposeDecrypt,
	}
)

// String returns a human-readable, string representation for a Purpose.
// Unrecognized purpose values are assumed to be PurposeVerify.
func (p Purpose) String() string {
	if value, ok := purposeMarshal[p]; ok {
		return value
	}

	return purposeMarshal[PurposeVerify]
}

func (p *Purpose) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		if unmarshalValue, ok := purposeUnmarshal[string(data[1:len(data)-1])]; ok {
			*p = unmarshalValue
			return nil
		}
	}

	return fmt.Errorf("Invalid key purpose: %s", data)
}

func (p Purpose) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\"")
	buffer.WriteString(p.String())
	buffer.WriteString("\"")

	return buffer.Bytes(), nil
}

// ParseKey handles parsing a key based on its purpose.  Sign and encrypt
// keys must be RSA private keys, while verify and decrypt keys must be
// RSA public keys.  For unknown purpose values, the key purpose is assumed
// to be verify.
func (p Purpose) ParseKey(pemKey []byte) (interface{}, error) {
	switch p {
	case PurposeSign:
		return crypto.ParseRSAPrivateKeyFromPEM(pemKey)

	case PurposeVerify:
		return crypto.ParseRSAPublicKeyFromPEM(pemKey)

	case PurposeEncrypt:
		return crypto.ParseRSAPrivateKeyFromPEM(pemKey)

	case PurposeDecrypt:
		return crypto.ParseRSAPublicKeyFromPEM(pemKey)

	default:
		return crypto.ParseRSAPublicKeyFromPEM(pemKey)
	}
}
