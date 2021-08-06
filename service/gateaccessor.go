package service

import (
	"errors"

	"github.com/xmidt-org/webpa-common/v2/xhttp/gate"
)

var errGateClosed = errors.New("gate is closed")

type gateAccessor struct {
	Gate     gate.Interface
	Accessor Accessor
}

func GateAccessor(g gate.Interface, accessor Accessor) Accessor {
	if g == nil {
		g = gate.New(true)
	}
	if accessor == nil {
		accessor = EmptyAccessor()
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
