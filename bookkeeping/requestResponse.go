package bookkeeping

import (
	"github.com/Comcast/webpa-common/xhttp"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"strings"
)

func Code(response *http.Response) []interface{} {
	return []interface{}{"code", response.StatusCode}
}
func Path(request *http.Request) []interface{} {
	return []interface{}{"path", request.URL.Path}
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

func ResponseBody(response *http.Response) []interface{} {
	if response.Body == nil {
		return []interface{}{"res-body", "empty body"}
	}
	body, getBody, err := xhttp.NewRewind(response.Body)
	if err != nil {
		return []interface{}{}
	}
	readCloser, err := getBody()
	if err != nil {
		return []interface{}{}
	}
	data, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		return []interface{}{}
	}
	response.Body = readCloser
	return []interface{}{"res-body", string(data)}

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
