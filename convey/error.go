package convey

type Compliance int

const (
	Full Compliance = iota
	Missing
	Invalid

	MissingFields
)

func (c Compliance) String() string {
	switch c {
	case Full:
		return "full"
	case Missing:
		return "missing-convey"
	case Invalid:
		return "invalid-convey"
	case MissingFields:
		return "convey-missing-fields"
	default:
		return "*invalid*"
	}
}

type Comply interface {
	Compliance() Compliance
}

func GetCompliance(err error) Compliance {
	if c, ok := err.(Comply); ok {
		return c.Compliance()
	}

	return Invalid
}

type Error struct {
	Err error
	C   Compliance
}

func (e Error) Error() string {
	return e.Err.Error()
}

func (e Error) Compliance() Compliance {
	return e.C
}
