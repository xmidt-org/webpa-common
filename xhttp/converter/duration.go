package converter

import (
	"reflect"
	"time"
)

func Duration(v string) reflect.Value {
	d, err := time.ParseDuration(v)
	if err != nil {
		return reflect.Value{}
	}

	return reflect.ValueOf(d)
}
