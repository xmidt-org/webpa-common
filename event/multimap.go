// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package event

import "fmt"

// MultiMap describes a set of events together with how those events should be processed.  Most often,
// event types are mapped to URLs, but that is not required.  Event values can be any string that is
// meaningful to an application.
type MultiMap map[string][]string

// Add appends one or more values to an event type.  If mappedTo is empty, this method does nothing.
// If the eventType doesn't exist, it is created.
func (m MultiMap) Add(eventType string, mappedTo ...string) {
	if len(mappedTo) == 0 {
		return
	}

	m[eventType] = append(m[eventType], mappedTo...)
}

// Set changes the given eventType so that it maps only to the values supplied in mappedTo.  If
// mappedTo is empty, this method deletes the event type.
func (m MultiMap) Set(eventType string, mappedTo ...string) {
	if len(mappedTo) == 0 {
		delete(m, eventType)
		return
	}

	copyOf := make([]string, len(mappedTo))
	copy(copyOf, mappedTo)
	m[eventType] = copyOf
}

// Get returns the values associated with the given event type.  The fallback event types, if supplied, are used
// if no values are present for the given eventType.  The fallback is useful for defaults, e.g. m.Get("IOT", "default").
func (m MultiMap) Get(eventType string, fallback ...string) ([]string, bool) {
	values, ok := m[eventType]
	if !ok {
		for i := 0; i < len(fallback) && !ok; i++ {
			values, ok = m[fallback[i]]
		}
	}

	return values, ok
}

// NestedToMultiMap translates a map with potentially nested string keys into a MultiMap.  This function is useful
// when unmarshalling from libraries that impose some meaning on a separator, like viper does with periods.  Essentially,
// this function returns a MultiMap that is the result of "flattening" the given raw map.
//
// The separator string must be nonempty.  It is used as the separator for nested map keys, e.g. "foo.bar".
func NestedToMultiMap(separator string, raw map[string]interface{}) (MultiMap, error) {
	if len(separator) == 0 {
		return nil, fmt.Errorf("The separator cannot be empty")
	}

	output := make(MultiMap, len(raw))
	if err := nestedToMultiMap("", separator, raw, output); err != nil {
		return nil, err
	}

	return output, nil
}

// nestedToMultiMap is a recursive function that builds a MultiMap by travsersing any nested maps with the raw map.
func nestedToMultiMap(base, separator string, raw map[string]interface{}, output MultiMap) error {
	var eventType string
	for k, v := range raw {
		if len(base) > 0 {
			eventType = base + separator + k
		} else {
			eventType = k
		}

		switch value := v.(type) {
		case string:
			output.Set(eventType, value)

		case []string:
			output.Set(eventType, value...)

		case []interface{}:
			for _, rawElement := range value {
				if stringElement, ok := rawElement.(string); ok {
					output.Add(eventType, stringElement)
				} else {
					return fmt.Errorf("Invalid element value of type %T: %v", v, v)
				}
			}

		case map[string]interface{}:
			if err := nestedToMultiMap(eventType, separator, value, output); err != nil {
				return err
			}

		case map[string][]string:
			for nestedKey, nestedValues := range value {
				output.Set(eventType+separator+nestedKey, nestedValues...)
			}

		default:
			return fmt.Errorf("Invalid raw event value of type %T: %v", v, v)
		}
	}

	return nil
}
