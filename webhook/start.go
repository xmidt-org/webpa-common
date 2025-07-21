// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type token string

type StartConfig struct {
	// maximum time allowed to wait for data to be retrieved
	Duration time.Duration `json:"duration"`

	// path to query for current hooks
	ApiPath string `json:"apiPath"`

	AuthHeader string `json:"authHeader"`

	// sat configuration data for requesting token
	Sat struct {
		// url path
		Path string `json:"path"`

		// client id
		Id string `json:"id"`

		// client secret
		Secret string `json:"secret"`

		// client capabilities
		Capabilities string `json:"capabilities"`

		// the obtained sat token
		Token token
	} `json:"sat"`

	// client is here for testing purposes
	client http.Client
}

type Result struct {
	Hooks []W
	Error error
}

type satReqResp struct {
	Token token `json:"serviceAccessToken"`
}

func NewStartFactory(v *viper.Viper) (sc *StartConfig) {
	if v == nil {
		v = viper.New()
		v.SetDefault("duration", 1000000000)
		v.SetDefault("apiPath", "http://111.2.3.44:5555/api")
		v.SetDefault("sat.path", "http://111.22.33.4.7777/sat")
		v.SetDefault("sat.id", "myidisthisstring")
		v.SetDefault("sat.secret", "donottellsecrets")
		v.SetDefault("sat.capabilities", "capabilitiesgohere")
	}
	v.Unmarshal(&sc)

	sc.client = http.Client{}

	return sc
}

func (sc *StartConfig) getAuthorization() (err error) {
	u, err := url.Parse(sc.Sat.Path)
	if err != nil {
		return
	}

	if sc.Duration > 0 {
		u.RawQuery = fmt.Sprintf("ttl=%d&capabilities=%s", int(sc.Duration.Seconds()), sc.Sat.Capabilities)
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Client-Id", sc.Sat.Id)
	req.Header.Set("X-Client-Secret", sc.Sat.Secret)

	resp, err := sc.client.Do(req)
	if err != nil {
		return
	}

	body, err := getPayload(resp)
	if err != nil {
		return
	}

	var srr satReqResp
	err = json.Unmarshal(body, &srr)
	if err != nil {
		return
	}
	sc.Sat.Token = srr.Token

	return
}

func getPayload(resp *http.Response) (body []byte, err error) {
	if resp == nil {
		return body, errors.New("response was nil")
	} else if resp.StatusCode >= 400 {
		return body, fmt.Errorf("response status code: %d", resp.StatusCode)
	} else {
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return body, fmt.Errorf("response body read failed. %v", err)
		}
		return
	}
}

func (sc *StartConfig) makeRequest() (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", sc.ApiPath, nil)
	if err != nil {
		return
	}
	req.Header.Set("content-type", "application/json")

	if len(sc.Sat.Token) < 1 && len(sc.AuthHeader) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", sc.AuthHeader))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.Sat.Token))
	}

	resp, err = sc.client.Do(req)
	if err != nil {
		return
	}

	return
}

func (sc *StartConfig) GetCurrentSystemsHooks(rc chan Result) {
	var hooks []W

	if sc.Sat.Token == "" {
		err := sc.getAuthorization()
		if err != nil {
			rc <- Result{hooks, err}
			return
		}
	}

	fn := func(sc *StartConfig, rChan chan Result) {
		// TODO why are we ignoring this error?
		resp, _ := sc.makeRequest()
		// TODO why are we ignoring this error?
		body, _ := getPayload(resp)
		err := json.Unmarshal(body, &hooks)

		// temporary fix to convert old webhook struct to new.
		if err != nil && strings.HasPrefix(err.Error(), "parsing time") {
			hooks, err = convertOldHooksToNewHooks(body)
		}

		rChan <- Result{hooks, err}
	}

	getHooksChan := make(chan Result, 1)
	timeout := time.After(sc.Duration)

	fn(sc, getHooksChan)
	for {
		select {
		case r := <-getHooksChan:

			if r.Error != nil || len(r.Hooks) <= 0 {
				time.Sleep(time.Second * 2) // wait a moment between queries
				fn(sc, getHooksChan)
			} else {
				rc <- Result{r.Hooks, r.Error}
				return
			}

		case <-timeout:
			rc <- Result{hooks, errors.New("Unable to obtain hook list in allotted time.")}
			return
		}
	}
}
