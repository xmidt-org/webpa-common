// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"encoding/json"
	"fmt"
	"io"
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
	var items = []*W{}
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

// update is an api call to processes a listenener registration for adding and updating
func (r *Registry) UpdateRegistry(rw http.ResponseWriter, req *http.Request) {
	payload, _ := io.ReadAll(req.Body)
	req.Body.Close()

	w, err := NewW(payload, req.RemoteAddr)
	if err != nil {
		jsonResponse(rw, http.StatusBadRequest, err.Error())
		return
	}

	s, err := json.Marshal(w)
	if err != nil {
		jsonResponse(rw, http.StatusInternalServerError, err.Error())
		return
	}

	err = r.m.Notifier.PublishMessage(string(s))
	if err != nil {
		jsonResponse(rw, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(rw, http.StatusOK, "Success")
}
