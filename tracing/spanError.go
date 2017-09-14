package tracing

import "bytes"

// SpanError is a simple slice of Spans that implements error.  To be meaningful,
// at least (1) Span in the slice must have an error.
type SpanError []Span

func (se SpanError) String() string {
	return se.Error()
}

// Spans implements the Spanned interface, making it convenient for reflection
func (se SpanError) Spans() []Span {
	return se
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

// Spans provides an abstract way to obtain any spans associated with an object,
// typically an error
func Spans(err interface{}) []Span {
	if spanned, ok := err.(Spanned); ok {
		return spanned.Spans()
	}

	return nil
}
