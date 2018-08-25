package xhttp

import "net/http"

// Client is an interface implemented by net/http.Client
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

var _ Client = (*http.Client)(nil)
