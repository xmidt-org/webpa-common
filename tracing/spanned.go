package tracing

// Spanned can be implemented by message objects to describe the spans
// involved in producing the message.  Generally, this interface should
// be implemented on transient objects that pass through the layers
// of an application.
type Spanned interface {
	Spans() []Span
}

// Mergeable represents a Spanned which can be merged with other spans
type Mergeable interface {
	Spanned

	// WithSpans returns an instance of this object with the new Spans, possibly
	// merged into those returned by Spans.  This method should generally return
	// a shallow copy of itself with the new spans, to preserve immutability.
	WithSpans(...Span) interface{}
}

// Spans extracts the slice of Span instances from a container, if possible.
//
//   If container implements Spanned, then container.Spans() is returned with a true.
//   If container is a Span, a slice of that one element is returned with a true.
//   If container is a []Span, it's returned as is with a true.
//   Otherwise, this function returns nil, false.
func Spans(container interface{}) ([]Span, bool) {
	switch v := container.(type) {
	case Span:
		return []Span{v}, true
	case []Span:
		return v, true
	case Spanned:
		return v.Spans(), true
	default:
		return nil, false
	}
}

// MergeSpans attempts to merge the given spans into a container.  If container does not
// implement Mergeable, or if spans is empty, then this function returns container as is with a false.
// Otherwise, each element of spans is merged with container, and result of container.WithSpans is
// returned with a true.
//
// Similar to Spans, each element of spans may be of type Span, []Span, or Spanned.  Any other type is skipped without error.
func MergeSpans(container interface{}, spans ...interface{}) (interface{}, bool) {
	if len(spans) == 0 {
		return container, false
	}

	if mergeable, ok := container.(Mergeable); ok {
		var mergedSpans []Span

		for _, s := range spans {
			switch v := s.(type) {
			case Span:
				mergedSpans = append(mergedSpans, v)
			case []Span:
				mergedSpans = append(mergedSpans, v...)
			case Spanned:
				mergedSpans = append(mergedSpans, v.Spans()...)
			}
		}

		// we still don't want to merge if we wound up with nothing to merge
		if len(mergedSpans) == 0 {
			return container, false
		}

		// if there are existing spans, preserve order by appending the collected spans we
		// have so far.  also, allocate a copy to avoid polluting the spans of the original container.
		if existingSpans := mergeable.Spans(); len(existingSpans) > 0 {
			copyOf := make([]Span, len(existingSpans))
			copy(copyOf, existingSpans)
			mergedSpans = append(copyOf, mergedSpans...)
		}

		return mergeable.WithSpans(mergedSpans...), true
	}

	return container, false
}

// NopMergeable is just a Mergeable with no other state.  This is useful for tests.
type NopMergeable []Span

func (nm NopMergeable) Spans() []Span {
	return nm
}

func (nm NopMergeable) WithSpans(spans ...Span) interface{} {
	return NopMergeable(spans)
}
