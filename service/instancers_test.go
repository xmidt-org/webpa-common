// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"testing"

	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/assert"
)

func testInstancers(t *testing.T, is Instancers) {
	assert := assert.New(t)

	assert.Equal(0, is.Len())
	assert.False(is.Has("nosuch"))
	i, ok := is.Get("nosuch")
	assert.Nil(i)
	assert.False(ok)

	assert.NotPanics(func() { is.Stop() })

	var (
		child1 = new(MockInstancer)
		child2 = new(MockInstancer)
	)

	is.Set("child1", child1)
	assert.Equal(1, is.Len())

	assert.False(is.Has("nosuch"))
	i, ok = is.Get("nosuch")
	assert.Nil(i)
	assert.False(ok)

	i, ok = is.Get("child1")
	assert.Equal(child1, i)
	assert.True(ok)

	is.Set("child2", child2)
	assert.Equal(2, is.Len())

	assert.False(is.Has("nosuch"))
	i, ok = is.Get("nosuch")
	assert.Nil(i)
	assert.False(ok)

	i, ok = is.Get("child1")
	assert.Equal(child1, i)
	assert.True(ok)

	i, ok = is.Get("child2")
	assert.Equal(child2, i)
	assert.True(ok)

	child1.On("Stop").Once()
	child2.On("Stop").Once()
	assert.NotPanics(func() { is.Stop() })

	assert.Equal(
		map[string]sd.Instancer{
			"child1": child1,
			"child2": child2,
		},
		map[string]sd.Instancer(is.Copy()),
	)

	child1.AssertExpectations(t)
	child2.AssertExpectations(t)
}

func TestInstancers(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		testInstancers(t, Instancers{})
		testInstancers(t, Instancers{}.Copy())
	})

	t.Run("Nil", func(t *testing.T) {
		testInstancers(t, nil)
		testInstancers(t, Instancers(nil).Copy())
	})
}
