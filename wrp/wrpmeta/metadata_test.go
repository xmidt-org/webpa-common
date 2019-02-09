package wrpmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBuilderInitialFieldsPresent(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		source = SourceMap{
			"key1":   "value1",
			"key2":   123,
			"key3":   -17.6,
			"unused": "unused",
		}

		builder = NewBuilder()
	)

	require.NotNil(builder)

	metadata, allPresent := builder.Apply(source,
		Field{From: "key1"},
		Field{From: "key2", To: "/key2"},
		Field{From: "key3", To: "asdf", Default: "default"},
	).Set("key4", "value4").Add(map[string]string{"key5": "value5"}, true).Build()

	assert.Equal(
		map[string]string{
			"key1":  "value1",
			"/key2": "123",
			"asdf":  "-17.6",
			"key4":  "value4",
			"key5":  "value5",
		},
		metadata,
	)

	assert.True(allPresent)

	metadata, allPresent = builder.Add(map[string]string{"key6": "value6"}, false).Build()

	assert.Equal(
		map[string]string{
			"key1":  "value1",
			"/key2": "123",
			"asdf":  "-17.6",
			"key4":  "value4",
			"key5":  "value5",
			"key6":  "value6",
		},
		metadata,
	)

	assert.False(allPresent)

}

func testBuilderSomeInitialFieldsAbsent(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		source = SourceMap{
			"key1":   "value1",
			"key2":   123,
			"unused": "unused",
		}

		builder = NewBuilder()
	)

	require.NotNil(builder)

	metadata, allPresent := builder.Apply(source,
		Field{From: "key1"},
		Field{From: "key2", To: "/key2"},
		Field{From: "missing1", To: "/missing1", Default: "default"},
		Field{From: "missing2", To: "/missing2"},
	).Set("key4", "value4").Build()

	assert.Equal(
		map[string]string{
			"key1":      "value1",
			"/key2":     "123",
			"/missing1": "default",
			"key4":      "value4",
		},
		metadata,
	)

	assert.False(allPresent)

	metadata, allPresent = builder.Add(map[string]string{"key5": "value5"}, true).Build()

	assert.Equal(
		map[string]string{
			"key1":      "value1",
			"/key2":     "123",
			"/missing1": "default",
			"key4":      "value4",
			"key5":      "value5",
		},
		metadata,
	)

	assert.False(allPresent)
}

func TestBuilder(t *testing.T) {
	t.Run("InitialFieldsPresent", testBuilderInitialFieldsPresent)
	t.Run("SomeInitialFieldsAbsent", testBuilderSomeInitialFieldsAbsent)
}
