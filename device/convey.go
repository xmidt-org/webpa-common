package device

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/Comcast/webpa-common/httperror"
	"github.com/ugorji/go/codec"
	"net/http"
)

const (
	DefaultConveyHeader = "X-Webpa-Convey"
)

var (
	conveyHandle codec.Handle = &codec.JsonHandle{
		IntegerAsString: 'L',
	}
)

// Convey represents a block of JSON that should be transmitted
// with each outbound device HTTP request
type Convey struct {
	Decoded map[string]interface{}
	Encoded string
}

// ToRequestDefault invokes c.ToRequest with the DefaultConveyHeader
func (c *Convey) ToRequestDefault(request *http.Request) {
	c.ToRequest(DefaultConveyHeader, request)
}

// ToRequest adds this Convey to the given request.  The header will contain
// the base64-encoded JSON value of this Convey object.
func (c *Convey) ToRequest(headerName string, request *http.Request) {
	request.Header.Set(headerName, c.Encoded)
}

// ConveyParser represents the various ways to obtain a Convey instance.  Instances
// are safe for concurrent access.
type ConveyParser interface {
	FromValue(string) (*Convey, error)
	FromRequest(*http.Request) (*Convey, error)
}

// NewConveyParser produces a ConveyParser using the supplied configuration.  If headerName
// is empty, DefaultConveyHeader is used.  If encoding is nil, base64.StdEncoding is used.
func NewConveyParser(headerName string, encoding *base64.Encoding) ConveyParser {
	if len(headerName) == 0 {
		headerName = DefaultConveyHeader
	}

	if encoding == nil {
		encoding = base64.StdEncoding
	}

	return &conveyParser{
		headerName: headerName,
		missingHeaderError: httperror.New(
			fmt.Sprintf("Missing header: %s", headerName),
			http.StatusBadRequest,
			nil,
		),
		encoding: encoding,
	}
}

type conveyParser struct {
	headerName         string
	missingHeaderError error
	encoding           *base64.Encoding
}

func (p *conveyParser) FromValue(value string) (*Convey, error) {
	input := bytes.NewBufferString(value)
	decoder := codec.NewDecoder(
		base64.NewDecoder(p.encoding, input),
		conveyHandle,
	)

	convey := new(Convey)
	err := decoder.Decode(&convey.Decoded)
	if err != nil {
		return nil, err
	}

	convey.Encoded = value
	return convey, nil
}

func (p *conveyParser) FromRequest(request *http.Request) (*Convey, error) {
	value := request.Header.Get(p.headerName)
	if len(value) == 0 {
		return nil, p.missingHeaderError
	}

	return p.FromValue(value)
}
