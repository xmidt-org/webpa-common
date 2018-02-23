package xhttp

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

var errNotRewindable = errors.New("That request is not rewindable")

// ReadSeekerCloser combines the behavior of io.Reader, io.Seeker, and io.Closer.
// This package uses this interface for basic optimizations.
type ReadSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}

type closeAdapter struct {
	io.ReadSeeker
}

func (ca closeAdapter) Close() error {
	return nil
}

// NopCloser is an analog of ioutil.NopCloser.  This function preserves io.Seeker semantics in
// the returned instance.  Additionally, if rs already implements io.Closer, this function
// returns rs as is.
func NopCloser(rs io.ReadSeeker) ReadSeekerCloser {
	if rsc, ok := rs.(ReadSeekerCloser); ok {
		return rsc
	}

	return closeAdapter{rs}
}

// NewRewind extracts all remaining bytes from an io.Reader, then uses NewRewindableBytes
// to produce a body and a get body function.  If any error occurred during reading, that error
// is returned and the other return values will be nil.
//
// This function performs certain optimizations on the returned body and get body function.  If
// r implements io.Seeker, then a get body function that simply invokes Seek(0, 0) is used.
// Additionally, this function honors the case where r implements io.Closer, preserving its
// Close() semantics.
func NewRewind(r io.Reader) (io.ReadCloser, func() (io.ReadCloser, error), error) {
	if rs, ok := r.(io.ReadSeeker); ok {
		// no need to bother reading bytes
		rsc := NopCloser(rs)

		return rsc,
			func() (io.ReadCloser, error) {
				_, err := rsc.Seek(0, 0)
				return rsc, err
			}, nil
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	body, getBody := NewRewindBytes(b)
	return body, getBody, nil
}

// NewRewindBytes produces both an io.ReadCloser that returns the given bytes
// and a function that produces a new io.ReadCloser that returns those same bytes.
// Both return values from this function are appropriate for http.Request.Body and
// http.Request.GetBody, respectively.
func NewRewindBytes(b []byte) (io.ReadCloser, func() (io.ReadCloser, error)) {
	rsc := NopCloser(bytes.NewReader(b))
	return rsc,
		func() (io.ReadCloser, error) {
			_, err := rsc.Seek(0, 0)
			return rsc, err
		}
}

// EnsureRewindable configures the given request's contents to be restreamed in the event
// of a redirect or other arbitrary code that must resubmit a request.  If this function
// is successful, Rewind can be used to rewind the request.
//
// If a GetBody function is already present on the request, this function does nothing
// as the given request is already rewindable.  Additionally, if there is no Body on the request,
// this function does nothing as there's no body to rewind.
func EnsureRewindable(r *http.Request) error {
	if r.GetBody != nil || r.Body == nil {
		return nil
	}

	body, getBody, err := NewRewind(r.Body)
	if err != nil {
		return err
	}

	r.Body = body
	r.GetBody = getBody
	return nil
}

// Rewind prepares a request body to be replayed.  If a GetBody function is present,
// that function is invoked.  An error is returned if this function could not rewind the request.
func Rewind(r *http.Request) error {
	if r.GetBody != nil {
		b, err := r.GetBody()
		if err != nil {
			return err
		}

		r.Body = b
		return nil
	}

	if r.Body == nil {
		// this request has no body, so it is always "rewound"
		return nil
	}

	return errNotRewindable
}
