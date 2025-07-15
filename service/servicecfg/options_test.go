// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package servicecfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/v2/service"
	"github.com/xmidt-org/webpa-common/v2/service/accessor"
)

func testOptionsDefault(t *testing.T, o *Options) {
	assert := assert.New(t)
	assert.Equal(accessor.DefaultVnodeCount, o.vnodeCount())
	assert.False(o.disableFilter())
	assert.Equal(service.DefaultScheme, o.defaultScheme())
}

func testOptionsCustom(t *testing.T) {
	var (
		assert = assert.New(t)

		o = Options{
			VnodeCount:    345234,
			DisableFilter: true,
			DefaultScheme: "ftp",
		}
	)

	assert.Equal(345234, o.vnodeCount())
	assert.True(o.disableFilter())
	assert.Equal("ftp", o.defaultScheme())
}

func TestOptions(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		testOptionsDefault(t, nil)
		testOptionsDefault(t, new(Options))
	})

	t.Run("Custom", testOptionsCustom)
}
