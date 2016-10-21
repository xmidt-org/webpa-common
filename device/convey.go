package device

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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

	defaultConveyHandler = conveyHandler{
		headerName: DefaultConveyHeader,
		missingHeaderError: httperror.New(
			fmt.Sprintf("Missing header: %s", DefaultConveyHeader),
			http.StatusBadRequest,
			nil,
		),
		encoding: base64.StdEncoding,
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

// ConveyHandler represents the various ways to obtain a Convey instance.  Instances
// are safe for concurrent access.
type ConveyHandler interface {
	// FromValue parses a base64-encoded JSON string into a Convey value.
	FromValue(string) (*Convey, error)

	// FromRequest examines the request to extract the Convey value.
	FromRequest(*http.Request) (*Convey, error)

	// ToRequest inserts the appropriate metadata for the given Convey value
	// into the supplied request.  This method is useful for clients when creating
	// an HTTP request to connect to a device Manager.
	ToRequest(*Convey, *http.Request)
}

// DefaultConveyHandler returns the default ConveyHandler instance.
func DefaultConveyHandler() ConveyHandler {
	return &defaultConveyHandler
}

// NewConveyHandler produces a ConveyHandler using the supplied configuration.  If headerName
// is empty, DefaultConveyHeader is used.  If encoding is nil, base64.StdEncoding is used.
//
// This function will simply return DefaultConveyHandler if headername is empty and encoding is nil.
func NewConveyHandler(headerName string, encoding *base64.Encoding) ConveyHandler {
	if len(headerName) == 0 && encoding == nil {
		// slight optimization: just return the default instead of creating a new instance
		return &defaultConveyHandler
	}

	handler := defaultConveyHandler
	if len(headerName) > 0 {
		handler.headerName = headerName
		handler.missingHeaderError = httperror.New(
			fmt.Sprintf("Missing header: %s", headerName),
			http.StatusBadRequest,
			nil,
		)
	}

	if encoding != nil {
		handler.encoding = encoding
	}

	return &handler
}

type conveyHandler struct {
	headerName         string
	missingHeaderError error
	encoding           *base64.Encoding
}

func (h *conveyHandler) FromValue(value string) (*Convey, error) {
	input := bytes.NewBufferString(value)
	decoder := codec.NewDecoder(
		base64.NewDecoder(h.encoding, input),
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

func (h *conveyHandler) FromRequest(request *http.Request) (*Convey, error) {
	value := request.Header.Get(h.headerName)
	if len(value) == 0 {
		return nil, h.missingHeaderError
	}

	return h.FromValue(value)
}

func (h *conveyHandler) ToRequest(convey *Convey, request *http.Request) {
	if convey != nil && len(convey.encoded) > 0 {
		request.Header.Set(h.headerName, convey.encoded)
	}
}
