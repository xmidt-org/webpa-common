package bookkeeping

import (
	"github.com/Comcast/webpa-common/xhttp"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"strings"
)

func Code(response CapturedResponse) []interface{} {
	return []interface{}{"code", response.Code}
}

func RequestBody(request *http.Request) []interface{} {
	err := xhttp.EnsureRewindable(request)
	if err != nil {
		return []interface{}{}
	}
	data, err := ioutil.ReadAll(request.Body)
	request.Body.Close()
	if err != nil {
		return []interface{}{}
	}
	xhttp.Rewind(request)
	if len(data) == 0 {
		return []interface{}{"req-body", "empty body"}
	}
	return []interface{}{"req-body", string(data)}

}

func ResponseBody(response CapturedResponse) []interface{} {
	if response.Payload == nil {
		return []interface{}{"res-body", "empty body"}
	}

	return []interface{}{"res-body", string(response.Payload)}

}

func RequestHeaders(headers ...string) RequestFunc {
	canonicalizedHeaders := getCanoicalizedHeaders(headers...)
	return func(request *http.Request) []interface{} {
		return parseHeader(request.Header, canonicalizedHeaders)
	}
}

func ResponseHeaders(headers ...string) ResponseFunc {
	canonicalizedHeaders := getCanoicalizedHeaders(headers...)
	return func(response CapturedResponse) []interface{} {
		return parseHeader(response.Header, canonicalizedHeaders)
	}
}

func getCanoicalizedHeaders(headers ...string) []string {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}
	return canonicalizedHeaders
}

func parseHeader(header http.Header, canonicalizedHeaders []string) []interface{} {
	kv := make([]interface{}, 0)
	for _, key := range canonicalizedHeaders {
		if values := header[key]; len(values) > 0 {
			kv = append(kv, key, values)
		}
	}
	return kv
}

func parseHeaderWithPrefix(header http.Header, canonicalizedHeaders []string) []interface{} {
	kv := make([]interface{}, 0)
	for _, prefix := range canonicalizedHeaders {
		for key, results := range header {
			if strings.HasPrefix(key, prefix) && len(results) > 0 {
				kv = append(kv, key, results)
			}
		}
	}
	return kv
}

func RequestHeadersWithPrefix(headers ...string) RequestFunc {
	canonicalizedHeaders := getCanoicalizedHeaders(headers...)
	return func(request *http.Request) []interface{} {
		if request == nil {
			return []interface{}{}
		}
		return parseHeaderWithPrefix(request.Header, canonicalizedHeaders)
	}
}

func ResponseHeadersWithPrefix(headers ...string) ResponseFunc {
	canonicalizedHeaders := getCanoicalizedHeaders(headers...)
	return func(response CapturedResponse) []interface{} {
		return parseHeaderWithPrefix(response.Header, canonicalizedHeaders)
	}
}
