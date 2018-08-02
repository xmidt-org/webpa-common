package gate

import (
	"fmt"
	"net/http"
)

// Status is an http.Handler that reports the status of a gate
type Status struct {
	Gate Interface
}

func (s *Status) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(response, `{"open": %t}`, s.Gate.Open())
	response.WriteHeader(http.StatusOK)
}
