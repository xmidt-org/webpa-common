package tracinghttp

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/Comcast/webpa-common/tracing"
	gokithttp "github.com/go-kit/kit/transport/http"
)

const (
	SpanHeader  = "X-Xmidt-Span"
	ErrorHeader = "X-Xmidt-Error"
)

// HeadersForSpans emits header information for each Span.  The timeLayout may be empty, in which case time.RFC3339 is used.
// All times are converted to UTC prior to formatting.
func HeadersForSpans(spans []tracing.Span, timeLayout string, h http.Header) {
	if len(timeLayout) == 0 {
		timeLayout = time.RFC3339
	}

	output := new(bytes.Buffer)
	for _, s := range spans {
		output.Reset()
		fmt.Fprintf(output, `"%s","%s","%s"`, s.Name(), s.Start().UTC().Format(timeLayout), s.Duration())
		h.Add(SpanHeader, output.String())

		if err := s.Error(); err != nil {
			output.Reset()
			if coder, ok := err.(gokithttp.StatusCoder); ok {
				fmt.Fprintf(output, `"%s",%d,"%s"`, s.Name(), coder.StatusCode(), err.Error())
			} else {
				fmt.Fprintf(output, `"%s",,"%s"`, s.Name(), err.Error())
			}

			h.Add(ErrorHeader, output.String())
		}
	}
}
