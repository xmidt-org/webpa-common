package key

import (
	"net/url"
	"testing"
)

func TestKeyURL(t *testing.T) {
	var supportedKeyURLs = []string{
		"file:///key.pem",
		"file:///etc/appname/key.pem",
		"file:///~/.ssh/mykey.pem",
		"http://foobar.com/directory/key.txt",
		"http://foobar.com/directory/key?q=value",
		"https://testit.com/another/key",
	}

	var unsupportedKeyURLs = []string{
		"ftp://foobar.com/directory/key.txt",
	}

	for _, keyURL := range supportedKeyURLs {
		expectedURL, err := url.Parse(keyURL)
		if err != nil {
			t.Fatalf("Failed to parse URL: [%s]", keyURL)
		}

		if !IsSupportedKeyURL(expectedURL) {
			t.Errorf("Key URL [%s] should be supported", keyURL)
		}

		actualURL, err := ParseKeyURL(keyURL)
		if err != nil {
			t.Errorf("Key URL [%s] should be supported", keyURL)
		}

		if *expectedURL != *actualURL {
			t.Errorf("Expected URL [%s] but got [%s]", expectedURL, actualURL)
		}
	}

	for _, keyURL := range unsupportedKeyURLs {
		expectedURL, err := url.Parse(keyURL)
		if err != nil {
			t.Fatalf("Failed to parse URL: [%s]", keyURL)
		}

		if IsSupportedKeyURL(expectedURL) {
			t.Errorf("Key URL [%s] should NOT be supported", keyURL)
		}

		actualURL, err := ParseKeyURL(keyURL)
		if actualURL != nil || err == nil {
			t.Errorf("Key URL [%s] should NOT be supported", keyURL)
		}
	}
}
