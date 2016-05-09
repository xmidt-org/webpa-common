package golog

import (
	"errors"
	"fmt"
	"github.com/ian-kent/go-log/layout"
)

const (
	invalidLayoutPattern string = "Invalid layout pattern: %s"
)

// Pattern is a log4j-style pattern for use with go-log
type Pattern string

// ToLayout converts this golog.Pattern into a layout.Layout
func (p Pattern) ToLayout() layout.Layout {
	return layout.Pattern(string(p))
}

func (p Pattern) MarshalJSON() ([]byte, error) {
	return []byte(`"` + p + `"`), nil
}

func (p *Pattern) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		layoutValue := string(data[1 : len(data)-1])
		layout.Pattern(layoutValue) // used as a check of the format
		*p = Pattern(layoutValue)
		return nil
	}

	return errors.New(fmt.Sprintf(invalidLayoutPattern, data))
}
