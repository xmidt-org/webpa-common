// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatInstance(t *testing.T) {
	var (
		testData = []struct {
			scheme   string
			address  string
			port     int
			expected string
		}{
			{"http", "somehost.com", 8080, "http://somehost.com:8080"},
			{"http", "somehost.com", 80, "http://somehost.com"},
			{"", "somehost.com", 8080, "https://somehost.com:8080"},
			{"https", "somehost.com", 8080, "https://somehost.com:8080"},
			{"https", "somehost.com", 80, "https://somehost.com:80"},
			{"ftp", "somehost.com", 1234, "ftp://somehost.com:1234"},
			{"http", "default.net", 0, "http://default.net"},
			{"", "default.net", 0, "https://default.net"},
		}
	)

	for _, record := range testData {
		t.Run(fmt.Sprintf("%s,%s,%d", record.scheme, record.address, record.port), func(t *testing.T) {
			assert.Equal(
				t,
				record.expected,
				FormatInstance(record.scheme, record.address, record.port),
			)
		})
	}
}

func TestNormalizeInstance(t *testing.T) {
	var (
		testData = []struct {
			defaultScheme string
			instance      string
			expected      string
			expectsError  bool
		}{
			{"", "", "", true},
			{"", "     \t\n\r", "", true},
			{"", "blah:blah:blah", "blah:blah:blah", true},
			{"", " blah:blah:blah ", "blah:blah:blah", true},
			{"", "somehost.com:8080", "https://somehost.com:8080", false},
			{"", " somehost.com:8080 ", "https://somehost.com:8080", false},
			{"http", "somehost.com:8080", "http://somehost.com:8080", false},
			{"http", " somehost.com:8080 ", "http://somehost.com:8080", false},
			{"https", "somehost.com:8080", "https://somehost.com:8080", false},
			{"https", " somehost.com:8080 ", "https://somehost.com:8080", false},
			{"ftp", "somehost.com:8080", "ftp://somehost.com:8080", false},
			{"ftp", " somehost.com:8080 ", "ftp://somehost.com:8080", false},
			{"", "http://foobar.com", "http://foobar.com", false},
			{"http", "http://foobar.com", "http://foobar.com", false},
			{"https", "http://foobar.com", "http://foobar.com", false},
		}
	)

	for _, record := range testData {
		t.Run(fmt.Sprintf("%s,%s", record.defaultScheme, record.instance), func(t *testing.T) {
			var (
				assert      = assert.New(t)
				actual, err = NormalizeInstance(record.defaultScheme, record.instance)
			)

			assert.Equal(record.expected, actual)
			if record.expectsError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
