// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"testing"
	"time"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewString(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		g1 = New(true)
		g2 = New(false)
	)

	require.NotNil(g1)
	require.NotNil(g2)
	// nolint: typecheck
	assert.NotEqual(g2.String(), g1.String())
}

func testNewInitiallyOpen(t *testing.T, g Interface) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedTimestamp = time.Now()
	)

	g.(*gate).now = func() time.Time { return expectedTimestamp }

	require.NotNil(g)
	assert.True(g.Open())
	state, initialTimestamp := g.State()
	assert.True(state)
	assert.False(initialTimestamp.IsZero())

	assert.False(g.Raise())
	assert.True(g.Open())
	state, actualTimestamp := g.State()
	assert.True(state)
	assert.Equal(initialTimestamp, actualTimestamp)

	assert.True(g.Lower())
	assert.False(g.Open())
	state, actualTimestamp = g.State()
	assert.False(state)
	assert.Equal(expectedTimestamp.UTC(), actualTimestamp)

	assert.False(g.Lower())
	assert.False(g.Open())
	state, actualTimestamp = g.State()
	assert.False(state)
	assert.Equal(expectedTimestamp.UTC(), actualTimestamp)

	assert.True(g.Raise())
	assert.True(g.Open())
	state, actualTimestamp = g.State()
	assert.True(state)
	assert.Equal(expectedTimestamp.UTC(), actualTimestamp)
}

func testNewInitiallyClosed(t *testing.T, g Interface) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedTimestamp = time.Now()
	)

	g.(*gate).now = func() time.Time { return expectedTimestamp }

	require.NotNil(g)
	assert.False(g.Open())
	state, initialTimestamp := g.State()
	assert.False(state)
	assert.False(initialTimestamp.IsZero())

	assert.False(g.Lower())
	assert.False(g.Open())
	state, actualTimestamp := g.State()
	assert.False(state)
	assert.Equal(initialTimestamp, actualTimestamp)

	assert.True(g.Raise())
	assert.True(g.Open())
	state, actualTimestamp = g.State()
	assert.True(state)
	assert.Equal(expectedTimestamp.UTC(), actualTimestamp)

	assert.False(g.Raise())
	assert.True(g.Open())
	state, actualTimestamp = g.State()
	assert.True(state)
	assert.Equal(expectedTimestamp.UTC(), actualTimestamp)

	assert.True(g.Lower())
	assert.False(g.Open())
	state, actualTimestamp = g.State()
	assert.False(state)
	assert.Equal(expectedTimestamp.UTC(), actualTimestamp)
}

func testNewInitiallyOpenWithGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		gauge = generic.NewGauge("test")
		g     = New(true, WithGauge(gauge))
	)

	require.NotNil(g)
	assert.True(g.Open())
	assert.Equal(Open, gauge.Value())

	assert.False(g.Raise())
	assert.True(g.Open())
	assert.Equal(Open, gauge.Value())

	assert.True(g.Lower())
	assert.False(g.Open())
	assert.Equal(Closed, gauge.Value())

	assert.False(g.Lower())
	assert.False(g.Open())
	assert.Equal(Closed, gauge.Value())

	assert.True(g.Raise())
	assert.True(g.Open())
	assert.Equal(Open, gauge.Value())
}

func testNewInitiallyClosedWithGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		gauge = generic.NewGauge("test")
		g     = New(false, WithGauge(gauge))
	)

	require.NotNil(g)
	assert.False(g.Open())
	assert.Equal(Closed, gauge.Value())

	assert.False(g.Lower())
	assert.False(g.Open())
	assert.Equal(Closed, gauge.Value())

	assert.True(g.Raise())
	assert.True(g.Open())
	assert.Equal(Open, gauge.Value())

	assert.False(g.Raise())
	assert.True(g.Open())
	assert.Equal(Open, gauge.Value())

	assert.True(g.Lower())
	assert.False(g.Open())
	assert.Equal(Closed, gauge.Value())
}

func TestNew(t *testing.T) {
	t.Run("String", testNewString)

	t.Run("InitiallyOpen", func(t *testing.T) {
		testNewInitiallyOpen(t, New(true))
		testNewInitiallyOpen(t, New(true, WithGauge(nil)))

		t.Run("WithGauge", testNewInitiallyOpenWithGauge)
	})

	t.Run("InitiallyClosed", func(t *testing.T) {
		testNewInitiallyClosed(t, New(false))
		testNewInitiallyClosed(t, New(false, WithGauge(nil)))

		t.Run("WithGauge", testNewInitiallyClosedWithGauge)
	})
}
