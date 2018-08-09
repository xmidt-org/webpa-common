package drain

import (
	"net/http"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/gorilla/schema"
)

type Start struct {
	Drainer Interface
}

func (s *Start) ServceHTTP(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	var (
		decoder = schema.NewDecoder()
		job     Job
	)

	if err := decoder.Decode(&job, request.PostForm); err != nil {
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if err := s.Drainer.Start(job); err != nil {
		xhttp.WriteError(response, http.StatusConflict, err)
	}
}
