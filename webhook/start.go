package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type token string

type StartConfig struct {
	// maximum time allowed to wait for data to be retrieved
	Duration time.Duration `json:"duration"`

	// path to query for current hooks
	ApiPath string `json:"apiPath"`

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
	expires int   `json:"expires_in"`
	Token   token `json:"serviceAccessToken"`
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
		return body, errors.New("Response was nil")
	} else if resp.StatusCode >= 400 {
		return body, errors.New(fmt.Sprintf("Response status code: %d", resp.StatusCode))
	} else {
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return body, errors.New(fmt.Sprintf("Response body read failed. %v", err))
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.Sat.Token))

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
		}
	}

	fn := func(sc *StartConfig, rChan chan Result) {
		resp, err := sc.makeRequest()
		body, err := getPayload(resp)
		err = json.Unmarshal(body, &hooks)

		// temporary fix to convert old webhook struct to new.
		if err != nil && strings.HasPrefix(err.Error(), "parsing time") {
			hooks, err = convertOldHooksToNewHooks(body)
		}

		rChan <- Result{hooks, err}
	}

	getHooksChan := make(chan Result, 1)
	ticker := time.NewTicker(sc.Duration)
	defer ticker.Stop()

	fn(sc, getHooksChan)
	for {
		select {
		case r := <-getHooksChan:

			if r.Error != nil || len(r.Hooks) <= 0 {
				time.Sleep(time.Second * 2)  // wait a moment between queries
				fn(sc, getHooksChan)
			} else {
				rc <- Result{r.Hooks, r.Error}
				return
			}

		case <-ticker.C:
			rc <- Result{hooks, errors.New("Unable to obtain hook list in allotted time.")}
			return
		}
	}
}
