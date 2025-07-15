// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilConstructor(t *testing.T) {
	var (
		assert = assert.New(t)
		next   = Constant{}
	)

	assert.Nil(NilConstructor(nil))
	assert.Equal(next, NilConstructor(next))
}
