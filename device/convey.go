package device

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ugorji/go/codec"
)

var (
	conveyHandle codec.Handle = &codec.JsonHandle{
		IntegerAsString: 'L',
	}
)

// Convey represents a block of JSON that should be transmitted
// with each outbound device HTTP request.  This type can marshal
// itself back into JSON supplying the original JSON object.
type Convey struct {
	decoded map[string]interface{}
	encoded string
}

func (c *Convey) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.decoded)
}

func (c *Convey) Encoded() string {
	return c.encoded
}

func (c *Convey) String() string {
	return fmt.Sprintf("%v", c.decoded)
}

func ParseConvey(value string) (*Convey, error) {
	input := bytes.NewBufferString(value)
	decoder := codec.NewDecoder(
		base64.NewDecoder(base64.StdEncoding, input),
		conveyHandle,
	)

	convey := new(Convey)
	err := decoder.Decode(&convey.decoded)
	if err != nil {
		return nil, err
	}

	convey.encoded = value
	return convey, nil
}
