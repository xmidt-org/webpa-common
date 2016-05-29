package resource

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Location represents a common type describing how to obtain an external resource.
type Location string

// Open returns a ReadCloser which can be used to read the bytes from the external resource.
// It is the caller's responsibility to close this resource.
//
// The resource string is interpreted simply:  If it begins with http:// or https://, an HTTP
// request is made to the location as a URL.  Otherwise, it is assumed to be a system file.
func (l Location) Open() (reader io.ReadCloser, err error) {
	defer func() {
		if reader != nil && err != nil {
			reader.Close()
			reader = nil
		}
	}()

	value := string(l)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		var response *http.Response
		response, err = http.Get(value)
		if response != nil {
			reader = response.Body
			if err == nil && response.StatusCode != 200 && response.StatusCode != 204 {
				err = fmt.Errorf(
					"Unable to access [%s]: server returned %s",
					l,
					response.Status,
				)
			}
		}
	} else {
		reader, err = os.Open(value)
	}

	return
}

// ReadAll is the resource analogue to ioutil.ReadAll: it reads all the bytes
// from the given location and returns them and any error.
func (l Location) ReadAll() ([]byte, error) {
	reader, err := l.Open()
	if err != nil {
		return nil, err
	}

	defer reader.Close()
	return ioutil.ReadAll(reader)
}
