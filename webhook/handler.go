package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
)

type Registry struct {
	m       *monitor
	Changes chan []W
}

func NewRegistry(mon *monitor) Registry {
	return Registry{
		m:       mon,
		Changes: mon.changes,
	}
}

// jsonResponse is an internal convenience function to write a json response
func jsonResponse(rw http.ResponseWriter, code int, msg string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	rw.Write([]byte(fmt.Sprintf(`{"message":"%s"}`, msg)))
}

// get is an api call to return all the registered listeners
func (r *Registry) GetRegistry(rw http.ResponseWriter, req *http.Request) {
	var items []*W
	for i := 0; i < r.m.list.Len(); i++ {
		items = append(items, r.m.list.Get(i))
	}

	if msg, err := json.Marshal(items); err != nil {
		jsonResponse(rw, http.StatusInternalServerError, err.Error())
	} else {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(msg)
	}
}

// parspIP returns just the ip address from "IP:port"
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

// registrationValidation checks W value requirements
func (w *W) registrationValidation() (string, int) {
	if w.Config.URL == "" {
		return "invalid Config URL", http.StatusBadRequest
	}
	if w.Config.ContentType == "" || w.Config.ContentType != "json" {
		return "invalid content_type", http.StatusBadRequest
	}
	if len(w.Matcher.DeviceId) == 0 {
		w.Matcher.DeviceId = []string{".*"} // match anything
	}
	if len(w.Events) == 0 {
		return "invalid events", http.StatusBadRequest
	}

	return "", http.StatusOK
}

// update is an api call to processes a listenener registration for adding and updating
func (r *Registry) UpdateRegistry(rw http.ResponseWriter, req *http.Request) {
	payload, err := ioutil.ReadAll(req.Body)
	req.Body.Close()

	w := new(W)
	err = json.Unmarshal(payload, w)
	if err != nil {
		jsonResponse(rw, http.StatusInternalServerError, err.Error())
		return
	}

	issue, code := w.registrationValidation()
	if issue != "" || code != http.StatusOK {
		jsonResponse(rw, code, issue)
		return
	}

	// update the requesters address
	ip, err := parseIP(req.RemoteAddr)
	if err != nil {
		jsonResponse(rw, http.StatusInternalServerError, err.Error())
		return
	}
	w.Address = ip

	// send W as a single item array
	msg, err := json.Marshal([1]W{*w})
	if err != nil {
		jsonResponse(rw, http.StatusInternalServerError, err.Error())
		return
	}

	r.m.Notifier.PublishMessage(string(msg))
}
