package xviper

type unmarshaler interface {
	Unmarshal(interface{}) error
}

func UnmarshalSeveral(u unmarshaler, v ...interface{}) error {
	var err error
	for i := 0; err == nil && i < len(v); i++ {
		err = u.Unmarshal(v[i])
	}

	return err
}

type defaulter interface {
	SetDefault(string, interface{})
}

type Defaults map[string]interface{}

func ApplyDefaults(d defaulter, v Defaults) {
	for key, value := range v {
		d.SetDefault(key, value)
	}
}
