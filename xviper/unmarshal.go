package xviper

// Unmarshaler describes the subset of Viper behavior dealing with unmarshaling into arbitrary values.
type Unmarshaler interface {
	Unmarshal(interface{}) error
}

// Unmarshal supplies a convenience for unmarshaling several values.  The first error
// encountered is returned, and any remaining values are not unmarshaled.
func Unmarshal(u Unmarshaler, v ...interface{}) error {
	var err error
	for i := 0; err == nil && i < len(v); i++ {
		err = u.Unmarshal(v[i])
	}

	return err
}

// MustUnmarshal is like Unmarshal, except that it panics when any error is encountered.
func MustUnmarshal(u Unmarshaler, v ...interface{}) {
	if err := Unmarshal(u, v...); err != nil {
		panic(err)
	}
}
