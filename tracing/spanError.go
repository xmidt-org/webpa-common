package tracing

import "bytes"

// SpanError is a simple slice of Spans that implements error.  To be meaningful,
// at least (1) Span in the slice must have an error.
type SpanError []Span

func (se SpanError) String() string {
	return se.Error()
}

func (se SpanError) Error() string {
	var output bytes.Buffer
	for _, s := range se {
		err := s.Error()
		if err != nil {
			if output.Len() > 0 {
				output.WriteRune(',')
			}

			output.WriteRune('"')
			output.WriteString(err.Error())
			output.WriteRune('"')
		}
	}

	return output.String()
}
