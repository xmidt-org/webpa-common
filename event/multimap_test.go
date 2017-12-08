package event

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMultiMapAdd(t *testing.T) {
	var (
		assert = assert.New(t)
		m      = MultiMap{
			"existing": []string{"value1"},
		}
	)

	m.Add("novalues")
	assert.NotContains(m, "novalues")

	m.Add("existing")
	assert.Equal([]string{"value1"}, m["existing"])

	m.Add("test", "foo")
	assert.Equal([]string{"foo"}, m["test"])

	m.Add("test", "bar")
	assert.Equal([]string{"foo", "bar"}, m["test"])
}

func testMultiMapSet(t *testing.T) {
	var (
		assert = assert.New(t)
		m      = MultiMap{
			"existing": []string{"value1"},
		}
	)

	m.Set("novalues")
	assert.NotContains(m, "novalues")

	m.Set("test", "foo")
	assert.Equal([]string{"foo"}, m["test"])

	m.Set("existing", "foo", "bar")
	assert.Equal([]string{"foo", "bar"}, m["existing"])
}

func testMultiMapGet(t *testing.T) {
	var (
		assert = assert.New(t)
		m      = MultiMap{
			"existing": []string{"value1"},
			"default":  []string{"another value"},
			"default2": []string{"graar", "shmaar"},
		}
	)

	value, ok := m.Get("nosuch")
	assert.Empty(value)
	assert.False(ok)

	value, ok = m.Get("nosuch", "another nosuch")
	assert.Empty(value)
	assert.False(ok)

	value, ok = m.Get("nosuch", "default")
	assert.Equal([]string{"another value"}, value)
	assert.True(ok)

	value, ok = m.Get("existing")
	assert.Equal([]string{"value1"}, value)
	assert.True(ok)

	value, ok = m.Get("existing", "nosuch")
	assert.Equal([]string{"value1"}, value)
	assert.True(ok)

	value, ok = m.Get("existing", "default")
	assert.Equal([]string{"value1"}, value)
	assert.True(ok)

	value, ok = m.Get("nosuch", "default")
	assert.Equal([]string{"another value"}, value)
	assert.True(ok)

	value, ok = m.Get("nosuch", "default2")
	assert.Equal([]string{"graar", "shmaar"}, value)
	assert.True(ok)

	value, ok = m.Get("nosuch", "still nosuch", "default2")
	assert.Equal([]string{"graar", "shmaar"}, value)
	assert.True(ok)
}

func TestMultiMap(t *testing.T) {
	t.Run("Add", testMultiMapAdd)
	t.Run("Set", testMultiMapSet)
	t.Run("Get", testMultiMapGet)
}

func testNestedToMultiMapRaw(s string, t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		raw = map[string]interface{}{
			"scalar":   "scalar value",
			"multi":    []string{"value1", "value2"},
			"rawMulti": []interface{}{"raw1", "raw2"},
			"simpleNested": map[string][]string{
				"single": []string{"value"},
				"double": []string{"value1", "value2"},
			},
			"complex": map[string]interface{}{
				"scalar":   "scalar value",
				"multi":    []string{"value1", "value2"},
				"rawMulti": []interface{}{"raw1", "raw2"},
				"simpleNested": map[string][]string{
					"single": []string{"value"},
					"double": []string{"value1", "value2"},
				},
				"deep": map[string]interface{}{
					"deeper": map[string]interface{}{
						"deepest": map[string]interface{}{
							"scalar":   "scalar value",
							"multi":    []string{"value1", "value2"},
							"rawMulti": []interface{}{"raw1", "raw2"},
							"simpleNested": map[string][]string{
								"single": []string{"value"},
								"double": []string{"value1", "value2"},
							},
						},
					},
				},
			},
		}

		expected = MultiMap{
			"scalar":   []string{"scalar value"},
			"multi":    []string{"value1", "value2"},
			"rawMulti": []string{"raw1", "raw2"},

			"simpleNested" + s + "single":                                                             []string{"value"},
			"simpleNested" + s + "double":                                                             []string{"value1", "value2"},
			"complex" + s + "scalar":                                                                  []string{"scalar value"},
			"complex" + s + "multi":                                                                   []string{"value1", "value2"},
			"complex" + s + "rawMulti":                                                                []string{"raw1", "raw2"},
			"complex" + s + "simpleNested" + s + "single":                                             []string{"value"},
			"complex" + s + "simpleNested" + s + "double":                                             []string{"value1", "value2"},
			"complex" + s + "deep" + s + "deeper" + s + "deepest" + s + "scalar":                      []string{"scalar value"},
			"complex" + s + "deep" + s + "deeper" + s + "deepest" + s + "multi":                       []string{"value1", "value2"},
			"complex" + s + "deep" + s + "deeper" + s + "deepest" + s + "rawMulti":                    []string{"raw1", "raw2"},
			"complex" + s + "deep" + s + "deeper" + s + "deepest" + s + "simpleNested" + s + "single": []string{"value"},
			"complex" + s + "deep" + s + "deeper" + s + "deepest" + s + "simpleNested" + s + "double": []string{"value1", "value2"},
		}
	)

	actual, err := NestedToMultiMap(s, raw)
	require.NotEmpty(actual)
	require.NoError(err)
	assert.Equal(expected, actual)
}

func testNestedToMultiMapViper(s string, t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		configuration = struct {
			Events map[string]interface{}
		}{
			Events: nil,
		}
	)

	v.SetConfigType("json")
	require.NoError(v.ReadConfig(strings.NewReader(`
	{ "events": {
		"foo": ["http://localhost/foo/bar"],
		"xfi.event.type": ["https://somewhere.comcast.net/bleh"],
		"another.event.type": ["http://1.1.1.1", "https://someplace.google.com"]
	}}
	`)))

	require.NoError(v.Unmarshal(&configuration))

	actual, err := NestedToMultiMap(s, configuration.Events)
	require.NotEmpty(actual)
	require.NoError(err)

	assert.Equal(
		MultiMap{
			"foo": []string{"http://localhost/foo/bar"},
			"xfi" + s + "event" + s + "type":     []string{"https://somewhere.comcast.net/bleh"},
			"another" + s + "event" + s + "type": []string{"http://1.1.1.1", "https://someplace.google.com"},
		},
		actual,
	)
}

func testNestedToMultiMapBadSeparator(t *testing.T) {
	assert := assert.New(t)
	m, err := NestedToMultiMap("", map[string]interface{}{})
	assert.Empty(m)
	assert.Error(err)
}

func testNestedToMultiMapBadEventValue(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []map[string]interface{}{
			map[string]interface{}{
				"bad": []interface{}{1234},
			},
			map[string]interface{}{
				"nested": map[string]interface{}{
					"bad": []interface{}{1234},
				},
			},
			map[string]interface{}{
				"bad": -17.6,
			},
		}
	)

	for _, bad := range testData {
		t.Logf("%#v", bad)
		m, err := NestedToMultiMap(".", bad)
		assert.Empty(m)
		assert.Error(err)
	}
}

func TestNestedToMultiMap(t *testing.T) {
	separators := []string{".", "-", " ", "asdf"}

	t.Run("Raw", func(t *testing.T) {
		for _, s := range separators {
			t.Run("Separator="+s, func(t *testing.T) {
				testNestedToMultiMapRaw(s, t)
			})
		}
	})

	t.Run("Viper", func(t *testing.T) {
		for _, s := range separators {
			t.Run("Separator="+s, func(t *testing.T) {
				testNestedToMultiMapViper(s, t)
			})
		}
	})

	t.Run("BadSeparator", testNestedToMultiMapBadSeparator)
	t.Run("BadEventValue", testNestedToMultiMapBadEventValue)
}
