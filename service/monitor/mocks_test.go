// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package monitor

import "github.com/stretchr/testify/mock"

type mockListener struct {
	mock.Mock
}

func (m *mockListener) MonitorEvent(e Event) {
	m.Called(e)
}
