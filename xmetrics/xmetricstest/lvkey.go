package xmetricstest

import (
	"bytes"
	"errors"
	"sort"
)

const (
	lvPairSeparator  = ','
	lvValueSeparator = '='
)

// Labeled provides access to a metric's labeled "children".
type Labeled interface {
	// Get returns the nested metric associated with a set of label/value pairs, if such a nested metric exists.
	// If the given key represents the root key, this same instance is returned.  The second return value will be
	// true if and only if the first is non-nil.
	Get(LVKey) (interface{}, bool)
}

var rootKey LVKey = ""

// LVKey represents a canonicalized key for a set of label/value pairs.
type LVKey string

func (lvk LVKey) Root() bool {
	return len(lvk) == 0
}

// NewLVKey produces a consistent, unique key for a set of label/value pairs.
// For example, {"code", "200", "method", "POST"} will result in the same key
// as {"method", "POST", "code", "200"}.
func NewLVKey(labelsAndValues []string) (LVKey, error) {
	count := len(labelsAndValues)
	if count == 0 {
		return rootKey, nil
	} else if count%2 != 0 {
		return rootKey, errors.New("Each label must be followed by a value")
	}

	var output bytes.Buffer
	switch count {
	case 2:
		// optimization: only a single pair, so no sorting to do
		output.WriteString(labelsAndValues[0])
		output.WriteRune(lvValueSeparator)
		output.WriteString(labelsAndValues[1])

	case 4:
		// optimization: 2 pairs, so we can just directly compare instead of
		// bothering with sorting
		a, b := 0, 2
		if labelsAndValues[a] > labelsAndValues[b] {
			a, b = b, a
		}

		output.WriteString(labelsAndValues[a])
		output.WriteRune(lvValueSeparator)
		output.WriteString(labelsAndValues[a+1])
		output.WriteRune(lvPairSeparator)
		output.WriteString(labelsAndValues[b])
		output.WriteRune(lvValueSeparator)
		output.WriteString(labelsAndValues[b+1])

	default:
		// we have 3 or more pairs, so go full hog and sort things
		var (
			labels = make([]string, 0, count/2)
			values = make(map[string]string, count/2)
		)

		for i := 0; i < count; i += 2 {
			labels = append(labels, labelsAndValues[i])
			values[labelsAndValues[i]] = labelsAndValues[i+1]
		}

		sort.Strings(labels)

		output.WriteString(labels[0])
		output.WriteRune(lvValueSeparator)
		output.WriteString(values[labels[0]])

		for i := 1; i < len(labels); i++ {
			output.WriteRune(lvPairSeparator)
			output.WriteString(labels[i])
			output.WriteRune(lvValueSeparator)
			output.WriteString(values[labels[i]])
		}
	}

	return LVKey(output.String()), nil
}
