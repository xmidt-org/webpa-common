// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiAccessorFactory(t *testing.T) {
	for _, v := range []int{-1, 0, 123, DefaultVnodeCount, 756} {
		t.Run(fmt.Sprintf("vnodeCount=%d", v), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				af = NewMultiAccessorFactory(Builder{MultiAccessor: []HasherConfig{{Type: HostnameType}}}, v)
			)

			require.NotNil(af)
			a := af([]string{"https://example.com"})
			require.NotNil(a)
			for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
				i, err := a.Get([]byte(k))
				assert.Equal([]string{"https://example.com"}, i)
				assert.NoError(err)
			}

		})
	}
}

func TestDefaultMultiAccessorFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		a = DefaultMultiAccessorFactory([]string{"https://example.com"})
	)

	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal([]string{"https://example.com"}, i)
		assert.NoError(err)
	}
}
