package service

import (
	"bytes"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/billhathaway/consistentHash"
	"net"
	"net/url"
	"sort"
	"strings"
)

// ParseHostPort parses a value of the form returned by net.JoinHostPort and
// produces a base URL.  If no scheme is present, this function prepends "http://"
// to the URL.  All base URLs returned by this function are guaranteed to have a
// scheme, host, and port.
//
// The go.serversets library returns endpoints in this format.  This function is
// used to turn and endpoint into a valid base URL for a given service.
func ParseHostPort(value string) (baseURL string, err error) {
	var host, portString string
	host, portString, err = net.SplitHostPort(value)
	if err != nil {
		return
	}

	if strings.Contains(host, "://") {
		baseURL = fmt.Sprintf("%s:%s", host, portString)
	} else {
		baseURL = fmt.Sprintf("http://%s:%s", host, portString)
	}

	return
}

// ReplaceHostPort accepts a hostPort value of the form produced by ParseHostPort and
// returns a URL with the scheme, host, and port replaced in the original URL.  The original
// URL's path, query, and fragment are preserved.
//
// This function is primarily useful when using a string returned from Accessor.Get to
// redirect to or dispatch to a hashed service node.
func ReplaceHostPort(hostPort string, originalURL *url.URL) string {
	var buffer bytes.Buffer
	buffer.WriteString(hostPort)

	path := originalURL.EscapedPath()
	if len(path) > 0 && path[0] != '/' {
		buffer.WriteByte('/')
	}

	buffer.WriteString(path)

	if originalURL.ForceQuery || len(originalURL.RawQuery) > 0 {
		buffer.WriteByte('?')
		buffer.WriteString(originalURL.RawQuery)
	}

	if len(originalURL.Fragment) > 0 {
		buffer.WriteByte('#')
		buffer.WriteString(originalURL.Fragment)
	}

	return buffer.String()
}

// Accessor provides access to services based around []byte keys.
// *consistentHash.ConsistentHash implements this interface.
type Accessor interface {
	Get([]byte) (string, error)
}

// AccessorFactory is a Factory Interface for creating service Accessors.
type AccessorFactory interface {
	// New creates an Accessor using a slice of endpoints.  Each endpoint must
	// be of the form parseable by ParseHostPort.  Invalid endpoints are skipped
	// with an error log message.  The returned slice of strings is the sorted
	// list of base URLs added to the Accessor.
	New([]string) (Accessor, []string)
}

// NewAccessorFactory uses a set of Options to produce an AccessorFactory
func NewAccessorFactory(o *Options) AccessorFactory {
	return &consistentHashFactory{
		logger:     o.logger(),
		vnodeCount: o.vnodeCount(),
	}
}

// consistentHashFactory creates consistentHash instances, which implement Accessor.
// This is the standard implementation of AccessorFactory.
type consistentHashFactory struct {
	logger     logging.Logger
	vnodeCount int
}

func (f *consistentHashFactory) New(endpoints []string) (Accessor, []string) {
	hash := consistentHash.New()
	hash.SetVnodeCount(f.vnodeCount)

	baseURLs := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if baseURL, err := ParseHostPort(endpoint); err != nil {
			f.logger.Error("Skipping bad endpoint: %s", endpoint)
		} else {
			baseURLs = append(baseURLs, baseURL)
		}
	}

	// sort first, before adding, to give a consistent ordering
	sort.Strings(baseURLs)
	for _, baseURL := range baseURLs {
		hash.Add(baseURL)
	}

	return hash, baseURLs
}
