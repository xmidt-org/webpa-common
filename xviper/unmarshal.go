package xviper

import "github.com/spf13/viper"

// Unmarshaler describes the subset of Viper behavior dealing with unmarshaling into arbitrary values.
type Unmarshaler interface {
	Unmarshal(interface{}, ...viper.DecoderConfigOption) error
}

// KeyUnmarshaler describes the subset of Viper behavior for unmarshaling an arbitrary configuration key.
type KeyUnmarshaler interface {
	UnmarshalKey(string, interface{}) error
}

// InvalidUnmarshaler is an Unmarshaler that simply returns an error.
// Mostly useful for testing.
type InvalidUnmarshaler struct {
	Err error
}

func (iu InvalidUnmarshaler) Unmarshal(interface{}, ...viper.DecoderConfigOption) error {
	return iu.Err
}

// MustUnmarshal attempts to unmarshal the value, panicing on any error.
func MustUnmarshal(u Unmarshaler, v interface{}) {
	if err := u.Unmarshal(v); err != nil {
		panic(err)
	}
}

// MustKeyUnmarshal attempts to unmarshal the given key, panicing on any error
func MustKeyUnmarshal(u KeyUnmarshaler, k string, v interface{}) {
	if err := u.UnmarshalKey(k, v); err != nil {
		panic(err)
	}
}
