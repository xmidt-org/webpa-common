package drain

import "net/http"

// Cancel is an HTTP handler that allows cancellation of drain jobs
type Cancel struct {
	Drainer Interface
}

func (c *Cancel) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	done, err := c.Drainer.Cancel()
	if err != nil {
		response.WriteHeader(http.StatusNotFound)
		return
	}

	select {
	case <-done:
	case <-request.Context().Done():
	}
}
