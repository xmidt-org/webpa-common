package drain

import (
	"encoding/json"
	"net/http"

	"github.com/Comcast/webpa-common/xhttp"
)

// Status returns a JSON message describing the status of the drain job
type Status struct {
	Drainer Interface
}

func (s *Status) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	var (
		active, job, progress = s.Drainer.Status()
		message, err          = json.Marshal(
			map[string]interface{}{
				"active":   active,
				"job":      job,
				"progress": progress,
			},
		)
	)

	if err != nil {
		xhttp.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	response.Write(message)
}
