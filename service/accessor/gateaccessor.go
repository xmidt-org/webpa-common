// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package accessor

import (
	"errors"

	"github.com/xmidt-org/webpa-common/v2/xhttp/gate"
)

var errGateClosed = errors.New("gate is closed")

type gateAccessor struct {
	Gate     gate.Interface
	Accessor Accessor
}

func GateAccessor(g gate.Interface, a Accessor) Accessor {
	if g == nil {
		g = gate.New(true)
	}
	if a == nil {
		a = EmptyAccessor()
	}
	return gateAccessor{
		Gate:     g,
		Accessor: a,
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
