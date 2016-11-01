package device

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
)

// Key is a routing identifier for a device.  While multiple devices in a Manager can have
// the same ID, Keys are unique to specific devices.
type Key string

// KeyFunc returns the unique Key for a device at the point of connection.
type KeyFunc func(ID, *Convey, *http.Request) (Key, error)

var (
	invalidKey = Key("")
)

// UUIDKeyFunc produces a KeyFunc which creates type 4 UUIDs for device Keys.
//
// If source == nil, then rand.Reader from crypto/rand is used.  If encoding == nil, then
// base64.RawURLEncoding is used.
func UUIDKeyFunc(source io.Reader, encoding *base64.Encoding) KeyFunc {
	if source == nil {
		source = rand.Reader
	}

	if encoding == nil {
		encoding = base64.RawURLEncoding
	}

	return func(ID, *Convey, *http.Request) (Key, error) {
		raw := make([]byte, 16)
		if _, err := source.Read(raw); err != nil {
			return invalidKey, err
		}

		raw[8] = (raw[8] | 0x80) & 0xBF
		raw[6] = (raw[6] | 0x40) & 0x4F

		output := new(bytes.Buffer)
		encoder := base64.NewEncoder(encoding, output)
		if _, err := encoder.Write(raw); err != nil {
			return invalidKey, err
		}

		if err := encoder.Close(); err != nil {
			return invalidKey, err
		}

		return Key(output.String()), nil
	}
}
