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

		jobMap = map[string]interface{}{
			"count": job.Count,
		}

		messageMap = map[string]interface{}{
			"active":   active,
			"progress": progress,
		}
	)

	// some specialness with the job to make it more human friendly
	if job.Percent > 0 {
		jobMap["percent"] = job.Percent
	}

	if job.Rate > 0 {
		jobMap["rate"] = job.Rate
	}

	if job.Tick > 0 {
		jobMap["tick"] = job.Tick.String()
	}

	messageMap["job"] = jobMap
	message, err := json.Marshal(messageMap)
	if err != nil {
		xhttp.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	response.Write(message)
}
