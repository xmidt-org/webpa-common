package tracinghttp

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/xmidt-org/webpa-common/tracing"
	"github.com/xmidt-org/webpa-common/xhttp"
	"github.com/stretchr/testify/assert"
)

func TestHeadersForSpans(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedStart    = time.Now()
		expectedDuration = time.Duration(2342123)
		spanner          = tracing.NewSpanner(
			tracing.Now(func() time.Time { return expectedStart }),
			tracing.Since(func(time.Time) time.Duration { return expectedDuration }),
		)

		testData = []struct {
			spans          []tracing.Span
			expectedHeader http.Header
			timeLayout     string
		}{
			{expectedHeader: http.Header{}},
			{
				spans: []tracing.Span{
					spanner.Start("first")(nil),
					spanner.Start("second")(errors.New("second error")),
					spanner.Start("third")(&xhttp.Error{Code: 503, Text: "fubar"}),
				},
				expectedHeader: http.Header{
					SpanHeader: []string{
						fmt.Sprintf(`"%s","%s","%s"`, "first", expectedStart.UTC().Format(time.RFC3339), expectedDuration.String()),
						fmt.Sprintf(`"%s","%s","%s"`, "second", expectedStart.UTC().Format(time.RFC3339), expectedDuration.String()),
						fmt.Sprintf(`"%s","%s","%s"`, "third", expectedStart.UTC().Format(time.RFC3339), expectedDuration.String()),
					},
					ErrorHeader: []string{
						`"second",,"second error"`,
						`"third",503,"fubar"`,
					},
				},
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		actualHeader := make(http.Header)
		HeadersForSpans(record.timeLayout, actualHeader, record.spans...)
		assert.Equal(record.expectedHeader, actualHeader)
	}
}
