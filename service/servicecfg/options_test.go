package servicecfg

import (
	"testing"

	"github.com/Comcast/webpa-common/service"
	"github.com/stretchr/testify/assert"
)

func testOptionsDefault(t *testing.T, o *Options) {
	assert := assert.New(t)
	assert.Equal(service.DefaultVnodeCount, o.vnodeCount())
	assert.False(o.disableFilter())
}

func testOptionsCustom(t *testing.T) {
	var (
		assert = assert.New(t)

		o = Options{
			VnodeCount:    345234,
			DisableFilter: true,
		}
	)

	assert.Equal(345234, o.vnodeCount())
	assert.True(o.disableFilter())
}

func TestOptions(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		testOptionsDefault(t, nil)
		testOptionsDefault(t, new(Options))
	})

	t.Run("Custom", testOptionsCustom)
}
