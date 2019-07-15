package servicecfg

import (
	"testing"

	"github.com/xmidt-org/webpa-common/service"
	"github.com/stretchr/testify/assert"
)

func testOptionsDefault(t *testing.T, o *Options) {
	assert := assert.New(t)
	assert.Equal(service.DefaultVnodeCount, o.vnodeCount())
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
