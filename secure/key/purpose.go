package key

import (
	"bytes"
	"errors"
	"fmt"
)

// Purpose is an enumerated type describing the reason a given
// key is being used
type Purpose int

const (
	PurposeSign      = Purpose(iota)
	PurposeSignValue = "sign"

	PurposeVerify      = Purpose(iota)
	PurposeVerifyValue = "verify"

	PurposeEncrypt      = Purpose(iota)
	PurposeEncryptValue = "encrypt"

	PurposeDecrypt      = Purpose(iota)
	PurposeDecryptValue = "decrypt"
)

var (
	// purposeUnmarshal is a reverse mapping of the string representations
	// for Purpose.  It's principally useful when unmarshalling values.
	purposeUnmarshal = map[string]Purpose{
		PurposeSignValue:    PurposeSign,
		PurposeVerifyValue:  PurposeVerify,
		PurposeEncryptValue: PurposeEncrypt,
		PurposeDecryptValue: PurposeDecrypt,
	}
)

// String returns a human-readable, string representation for a Purpose
func (Purpose Purpose) String() string {
	switch Purpose {
	default:
		return PurposeVerifyValue

	case PurposeSign:
		return PurposeSignValue

	case PurposeVerify:
		return PurposeVerifyValue

	case PurposeEncrypt:
		return PurposeEncryptValue

	case PurposeDecrypt:
		return PurposeDecryptValue
	}
}

func (purpose *Purpose) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		if unmarshalValue, ok := purposeUnmarshal[string(data[1:len(data)-1])]; ok {
			*purpose = unmarshalValue
			return nil
		}
	}

	return errors.New(fmt.Sprintf("Invalid key purpose: %s", data))
}

func (purpose Purpose) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\"")
	buffer.WriteString(purpose.String())
	buffer.WriteString("\"")

	return buffer.Bytes(), nil
}
