package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

const (
	DEFAULT_EXPIRATION_DURATION time.Duration = time.Second * 300
)

type Registry struct {
	UpdatableList
}

// getAll builds a list of registered listeners
func (r *Registry) getAll() (all []*W) {
	for i:=0; i<r.Len(); i++ {
		all = append(all, r.Get(i))
	}
	
	return
}

// getRegistered is an api call to return all the registered listeners
func (r *Registry) getRegistered(rw http.ResponseWriter, req *http.Request) {
	if json, err := json.Marshal( r.getAll() ); err != nil {
//		log.Error("JSON marshal hooks error %v", err.Error())
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln( rw, fmt.Sprintf(`{"message":"%s"}`, err.Error()) )
	} else {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(json)
	}
}


func parseIP(s string) (string, error) {
	ip1, _, err := net.SplitHostPort(s)
	if err == nil {
		return ip1, nil
	}

	ip2 := net.ParseIP(s)
	if ip2 == nil {
		return "", errors.New("invalid IP")
	}

	return ip2.String(), nil
}

func (w *W) registrationCheck() (string, int) {
	if w.URL == "" {
		return "invalid Config URL", http.StatusBadRequest
	}
	if w.ContentType == "" || w.ContentType != "json" {
		return "invalid content_type", http.StatusBadRequest
	}
	if len(w.Matcher.DeviceId) == 0 {
		w.Matcher.DeviceId = []string{".*"}  // match anything
	}	
	if len(w.Events) == 0 {
		return "invalid events", http.StatusBadRequest
	}

	return "", http.StatusOK
}

func (r *Registry) register(rw http.ResponseWriter, req *http.Request) {
	payload, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	
	w := new(W)
	err = json.Unmarshal(payload, w)
	if err != nil {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln( rw, fmt.Sprintf(`{"message":"%s"}`, err.Error()) )
		return
	}
	
	ip, err := parseIP(req.RemoteAddr)
	if err != nil {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln( rw, fmt.Sprintf(`{"message":"%s"}`, err.Error()) )
		return
	}
	w.Address = ip
	
	msg, code := w.registrationCheck()
	if msg != "" || code != http.StatusOK {
		rw.WriteHeader(code)
		fmt.Fprintln( rw, fmt.Sprintf(`{"message":"%s"}`, msg) )
		return
	}
	
	regListeners := r.getAll()
	found := false
	for i:=0; i<len(regListeners) && !found; i++ {
		if regListeners[i].URL == w.Address {
			found = true
			if w.Duration > 0 && w.Duration < DEFAULT_EXPIRATION_DURATION {
				regListeners[i].Until = time.Now().Unix() + w.Duration
			} else {
				regListeners[i].Until = time.Now().Unix() + int64(DEFAULT_EXPIRATION_DURATION.Seconds())
			}
			regListeners[i].Matcher = w.Matcher
			regListeners[i].Events = w.Events
			regListeners[i].Groups = w.Groups
			regListeners[i].Config.ContentType = w.ContentType
			regListeners[i].Config.Secret = w.Secret
		}
	}
	if !found {
		w.Until = time.Now().Unix() + int64(DEFAULT_EXPIRATION_DURATION.Seconds())
		regListeners := append(regListeners, w)
	}
	
	r.Update(regListeners)
	
	rw.Header().Set("Content-Type", "application/json")
	rw.Write([]byte(`{"message":"Success"}`))
}