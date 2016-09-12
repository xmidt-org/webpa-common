package key

import (
	"bytes"
	"fmt"
)

// Purpose is an enumerated type describing the reason a given
// key is being used.  This type implements Parser.
//
// All Purpose values assume PEM-encoded keys.  For other formats,
// a custom Parser decorator can be used.  Purpose.RequiresPrivateKey()
// determines whether to parse the key as a private key.
type Purpose int

const (
	// PurposeVerify refers to a key used to verify a signature.  This is the zero-value
	// for Purpose.  These keys must be public keys encoded as PEM blocks.
	PurposeVerify Purpose = Purpose(iota)

	// PurposeSign refers to a key used to create a signature.  These keys must be private,
	// PEM-encoded keys.
	PurposeSign

	// PurposeEncrypt refers to a key used to encrypt data.  These keys must be private,
	// PEM-encoded keys.
	PurposeEncrypt

	// PurposeDecrypt refers to a key used to decrypt data.  These keys must be public,
	// PEM-encoded keys.
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

// RequiresPrivateKey returns true if this purpose requires a private key,
// false if it requires a public key.
func (p Purpose) RequiresPrivateKey() bool {
	return p == PurposeSign || p == PurposeEncrypt
}
