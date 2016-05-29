package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	EmptyLocation = errors.New("The location cannot be empty")
)

// LoaderFactory provides an abstract way to product Loader instances
// from resource locations.  It also provides a JSON representation
// of a resource.
type LoaderFactory struct {
	location string
	buffer   []byte
}

func (lf *LoaderFactory) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		location := string(data[1 : len(data)-1])
		if len(location) > 0 {
			lf.location = location
			lf.buffer = nil
			return nil
		} else {
			return EmptyLocation
		}
	}

	buffer := struct {
		Buffer string `json:"buffer"`
	}{}

	if err := json.Unmarshal(data, &buffer); err == nil {
		lf.buffer = []byte(buffer.Buffer)
		lf.location = ""
		return nil
	} else {
		return fmt.Errorf("Unable to read resource location %s: %v", data, err)
	}
}

func (lf LoaderFactory) MarshalJSON() ([]byte, error) {
	if len(lf.buffer) == 0 {
		return []byte(`"` + lf.location + `"`), nil
	}

	buffer := struct {
		Buffer string `json:"buffer"`
	}{
		Buffer: string(lf.buffer),
	}

	return json.Marshal(buffer)
}

// NewLoader creates a new Loader instance using this factory's configuration
func (lf LoaderFactory) NewLoader() Loader {
	if len(lf.buffer) > 0 {
		return Buffer(lf.buffer)
	}

	if strings.HasPrefix(lf.location, "http://") || strings.HasPrefix(lf.location, "https://") {
		return URL(lf.location)
	}

	return File(lf.location)
}

// Buffer creates a new in-memory loader
func Buffer(content []byte) Loader {
	return &bufferLoader{content}
}

// URL creates a new Loader which uses the supplied URL
func URL(url string) Loader {
	return &urlLoader{url}
}

// File creates a new Loader which uses the supplied system file
func File(file string) Loader {
	return &fileLoader{file}
}
