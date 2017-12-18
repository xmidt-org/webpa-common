package xhttp

import (
	"fmt"
	"net/http"
	"net/textproto"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const (
	DefaultMaxRedirects = 10
)

// RedirectPolicy is the configurable policy for handling redirects
type RedirectPolicy struct {
	// Logger is the go-kit Logger used for logging.  If unset, logging.DefaultLogger() is used.
	Logger log.Logger

	// MaxRedirects is the maximum number of redirects to follow.  If unset, DefaultMaxRedirects is used.
	MaxRedirects int

	// ExcludeHeaders is the blacklist of headers that should not be copied from previous requests.
	ExcludeHeaders []string
}

// logger returns the go-kit logger for output
func (p RedirectPolicy) logger() log.Logger {
	if p.Logger != nil {
		return p.Logger
	}

	return logging.DefaultLogger()
}

// maxRedirects returns the maximum number of redirects to follow
func (p RedirectPolicy) maxRedirects() int {
	if p.MaxRedirects > 0 {
		return p.MaxRedirects
	}

	return DefaultMaxRedirects
}

// headerFilter returns a closure that returns true if a header name should be included in redirected requests
func (p RedirectPolicy) headerFilter() func(string) bool {
	if len(p.ExcludeHeaders) > 0 {
		excludes := make(map[string]bool, len(p.ExcludeHeaders))
		for _, v := range p.ExcludeHeaders {
			excludes[textproto.CanonicalMIMEHeaderKey(v)] = true
		}

		return func(h string) bool {
			return !excludes[h]
		}
	}

	return func(string) bool {
		return true
	}
}

// CheckRedirect produces a redirect policy function given a policy descriptor
func CheckRedirect(p RedirectPolicy) func(*http.Request, []*http.Request) error {
	var (
		logger       = p.logger()
		maxRedirects = p.maxRedirects()
		headerFilter = p.headerFilter()
	)

	return func(r *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			err := fmt.Errorf("stopped after %d redirect(s)", maxRedirects)
			logger.Log(level.Key(), level.ErrorValue(), logging.ErrorKey(), err)
			return err
		}

		for k, v := range via[len(via)-1].Header {
			if headerFilter(k) {
				r.Header[k] = v
			} else {
				logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "excluding header on redirect", "header", k)
			}
		}

		return nil
	}
}
