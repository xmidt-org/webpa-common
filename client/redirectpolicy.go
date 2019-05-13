package client

import (
	"net/http"
	"reflect"

	"github.com/Comcast/webpa-common/xhttp"
)

type RedirectPolicyConfig struct {
	// MaxRedirects defines the maximum number of redirects each fanout will allow
	MaxRedirects int `json:"maxRedirects,omitempty"`

	// RedirectExcludeHeaders are the headers that will *not* be copied on a redirect
	RedirectExcludeHeaders []string `json:"redirectExcludeHeaders,omitempty"`
}

func (c *RedirectPolicyConfig) checkRedirect() func(*http.Request, []*http.Request) error {
	return xhttp.CheckRedirect(xhttp.RedirectPolicy{
		MaxRedirects:   c.maxRedirects(),
		ExcludeHeaders: c.redirectExcludeHeaders(),
	})
}

func (c *RedirectPolicyConfig) maxRedirects() int {
	if c != nil && c.MaxRedirects > 0 {
		return c.MaxRedirects
	}

	return 0
}

func (c *RedirectPolicyConfig) redirectExcludeHeaders() []string {
	if c != nil && len(c.RedirectExcludeHeaders) < 0 {
		return c.RedirectExcludeHeaders
	}

	return nil
}

func (c *RedirectPolicyConfig) IsEmpty() bool {
	return reflect.DeepEqual(c, (RedirectPolicyConfig{}))
}
