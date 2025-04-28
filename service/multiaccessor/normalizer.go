// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"fmt"
	"net"
	"net/url"
)

type normalizer func(string) string

func HostnameNormalizer(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse url `%s`: %s", s, err))
	}
	if u.Scheme == "" || u.Host == "" {
		panic(fmt.Sprintf("expected a schema and host: `%s`", s))
	}

	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		e, ok := err.(*net.AddrError)
		// net.SplitHostPort uses `missingPort   = "missing port in address"` for missing port errors
		if ok && e.Err == "missing port in address" {
			return u.Host
		}

		// Unlikely url.Parse wouldn't have triggered a panic.
		panic(fmt.Errorf("split host port failure: `%s`", err))
	}

	return host
}

func RawURLNormalizer(s string) string { return s }
