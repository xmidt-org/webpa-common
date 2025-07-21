// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"fmt"
	"net/url"
)

// ApplyURLParser applies a given URL parser, such as url.Parse or url.ParseRequestURI, to zero or more strings.
// The resulting slice is ordered the same as the values.  Any error halts parsing of subsequent values.
func ApplyURLParser(parser func(string) (*url.URL, error), values ...string) ([]*url.URL, error) {
	urls := make([]*url.URL, len(values))
	for i, v := range values {
		u, err := parser(v)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse URL '%s': %s", v, err)
		}

		urls[i] = u
	}

	return urls, nil
}
