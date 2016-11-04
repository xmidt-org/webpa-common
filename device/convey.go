package device

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ugorji/go/codec"
	"reflect"
)

var (
	conveyHandle codec.Handle = &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				MapType: reflect.TypeOf(map[string]interface{}(nil)),
			},
		},
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

func NewConvey(decoded map[string]interface{}) (*Convey, error) {
	output := new(bytes.Buffer)
	encoder := codec.NewEncoder(
		base64.NewEncoder(base64.StdEncoding, output),
		conveyHandle,
	)

	if err := encoder.Encode(decoded); err != nil {
		return nil, err
	}

	return &Convey{
		decoded: decoded,
		encoded: output.String(),
	}, nil
}

func ParseConvey(encoded string) (*Convey, error) {
	input := bytes.NewBufferString(encoded)
	decoder := codec.NewDecoder(
		base64.NewDecoder(base64.StdEncoding, input),
		conveyHandle,
	)

	convey := new(Convey)
	if err := decoder.Decode(&convey.decoded); err != nil {
		return nil, err
	}

	convey.encoded = encoded
	return convey, nil
}
