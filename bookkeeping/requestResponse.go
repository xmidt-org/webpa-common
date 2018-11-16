package bookkeeping

import (
	"net/http"
	"net/textproto"
	"strings"
)

func Code() ResponseFunc {
	return func(response *http.Response) []interface{} {
		return []interface{}{"code", response.StatusCode}
	}
}
func Path() RequestFunc {
	return func(request *http.Request) []interface{} {
		return []interface{}{"path", request.URL.Path}
	}
}

func RequestBody() RequestFunc {
	return func(request *http.Request) []interface{} {

		return []interface{}{"body"}

	}
}

func ResponseBody() ResponseFunc {
	return func(response *http.Response) []interface{} {

		return []interface{}{}

	}
}

func RequestHeaders(headers ...string) RequestFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}
	return func(request *http.Request) []interface{} {
		kv := make([]interface{}, 0)
		header := request.Header
		for _, key := range canonicalizedHeaders {
			if values := header[key]; len(values) > 0 {
				kv = append(kv, key, values)
			}
		}
		return kv
	}
}

func ResponseHeaders(headers ...string) ResponseFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}
	return func(response *http.Response) []interface{} {
		kv := make([]interface{}, 0)
		header := response.Header
		for _, key := range canonicalizedHeaders {
			if values := header.Get(key); len(values) > 0 {
				kv = append(kv, key, values)
			}
		}
		return kv
	}
}

func RequestHeadersWithPrefix(headers ...string) RequestFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}
	return func(request *http.Request) []interface{} {
		if request == nil {
			return []interface{}{}
		}
		kv := make([]interface{}, 0)
		header := request.Header
		for _, prefix := range canonicalizedHeaders {
			for key, results := range header {

				if strings.HasPrefix(key, prefix) && len(results) > 0 {
					kv = append(kv, key, results)
				}
			}
		}
		return kv
	}
}

func ResponseHeadersWithPrefix(headers ...string) ResponseFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}
	return func(response *http.Response) []interface{} {
		kv := make([]interface{}, 0)
		header := response.Header
		for _, prefix := range canonicalizedHeaders {
			for key, results := range header {

				if strings.HasPrefix(key, prefix) && len(results) > 0 {
					kv = append(kv, key, results)
				}
			}
		}
		return kv
	}
}
