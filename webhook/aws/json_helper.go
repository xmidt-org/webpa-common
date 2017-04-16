package aws

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"fmt"
)

const (
	MEDIATYPE_JSON = "application/json"
)

var (
	ErrJsonEmpty   = errors.New("JSON payload is empty")
)

func DecodeJsonPayload(req *http.Request, v interface{}) ([]byte, error) {
	if req.Body == nil {
		return nil, ErrJsonEmpty
	}

	payload, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, ErrJsonEmpty
	}
	err = json.Unmarshal([]byte(payload), v)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func ResponseJson(v interface{}, rw http.ResponseWriter) {
	jsonStr, err := json.Marshal(v)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", MEDIATYPE_JSON)
	rw.Write(jsonStr)
	return
}

func ResponseJsonErr(rw http.ResponseWriter, errmsg string, code int) {
	rw.Header().Set("Content-Type", MEDIATYPE_JSON)
	jsonStr := fmt.Sprintf(`{"message":"%s"}`, errmsg)
	rw.WriteHeader(code)
	fmt.Fprintln(rw, jsonStr)
	return
}