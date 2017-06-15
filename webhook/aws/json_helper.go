package aws

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

var (
	ErrJsonEmpty = errors.New("JSON payload is empty")
)

func DecodeJSONMessage(req *http.Request, v interface{}) ([]byte, error) {

	payload, err := ioutil.ReadAll(req.Body)
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
