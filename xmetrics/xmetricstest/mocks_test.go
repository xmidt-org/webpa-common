// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xmetricstest

import "github.com/stretchr/testify/mock"

type mockTestingT struct {
	mock.Mock
}

func (m *mockTestingT) Errorf(msg string, v ...any) {
	m.Called(msg, v)
}

func AnyMessage(_ string) bool {
	return true
}

func AnyArguments(_ []any) bool {
	return true
}
