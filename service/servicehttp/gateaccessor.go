package servicehttp

import (
	"errors"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/xhttp/gate"
)

var errGateClosed = errors.New("gate is closed")

type gateAccessor struct {
	Gate     gate.Interface
	Accessor service.Accessor
}

func GateAccessor(g gate.Interface, accessor service.Accessor) service.Accessor {
	if g == nil {
		g = gate.New(true)
	}
	if accessor == nil {
		accessor = service.EmptyAccessor()
	}
	return gateAccessor{
		Gate:     g,
		Accessor: accessor,
	}
}

func (ga gateAccessor) Get(key []byte) (string, error) {
	instance, err := ga.Accessor.Get(key)
	if err != nil {
		return instance, err
	}
	if !ga.Gate.Open() {
		return instance, errGateClosed
	}
	return instance, nil
}
