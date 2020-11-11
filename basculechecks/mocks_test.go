/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package basculechecks

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/bascule"
)

type mockCapabilitiesChecker struct {
	mock.Mock
}

func (m *mockCapabilitiesChecker) Check(auth bascule.Authentication, v ParsedValues) (string, error) {
	args := m.Called(auth, v)
	return args.String(0), args.Error(1)
}
