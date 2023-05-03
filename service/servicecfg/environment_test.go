package servicecfg

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/service"
	"github.com/xmidt-org/webpa-common/v2/service/consul"
	"github.com/xmidt-org/webpa-common/v2/service/zk"
	"github.com/xmidt-org/webpa-common/v2/xviper"
	"go.uber.org/zap"
)

func testNewEnvironmentEmpty(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()
	)

	e, err := NewEnvironment(nil, v)
	assert.Nil(e)
	assert.Error(err)
}

func testNewEnvironmentUnmarshalError(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected unmarshal error")
		u             = xviper.InvalidUnmarshaler{Err: expectedError}
	)

	e, actualError := NewEnvironment(nil, u)
	assert.Nil(e)
	assert.Equal(expectedError, actualError)
}

func testNewEnvironmentFixed(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = sallust.Default()
		v      = viper.New()

		configuration = strings.NewReader(`
			{
				"fixed": ["instance1.com:1234", "instance2.net:8888"]
			}
		`)
	)

	v.SetConfigType("json")
	require.NoError(v.ReadConfig(configuration))

	e, err := NewEnvironment(logger, v)
	require.NoError(err)
	require.NotNil(e)

	i := e.Instancers()
	assert.Len(i, 1)
	assert.NotNil(i["fixed"])

	assert.NoError(e.Close())
}

func testNewEnvironmentZookeeper(t *testing.T) {
	defer resetEnvironmentFactories()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = sallust.Default()
		v      = viper.New()

		expectedEnvironment = service.NewEnvironment()

		configuration = strings.NewReader(`
			{
				"zookeeper": {
					"client": {
						"connection": "host1.com:1111,host2.com:2222",
						"connectTimeout": "10s",
						"sessionTimeout": "20s"
					},
					"watches": ["/some/where"]
				}
			}
		`)
	)

	v.SetConfigType("json")
	require.NoError(v.ReadConfig(configuration))

	zookeeperEnvironmentFactory = func(l *zap.Logger, zo zk.Options, eo ...service.Option) (service.Environment, error) {
		assert.Equal(logger, l)
		assert.Equal(
			zk.Options{
				Client: zk.Client{
					Connection:     "host1.com:1111,host2.com:2222",
					ConnectTimeout: 10 * time.Second,
					SessionTimeout: 20 * time.Second,
				},
				Watches: []string{"/some/where"},
			},
			zo,
		)

		return expectedEnvironment, nil
	}

	actualEnvironment, err := NewEnvironment(logger, v)
	require.NoError(err)
	require.NotNil(actualEnvironment)
	assert.Equal(expectedEnvironment, actualEnvironment)

	assert.NoError(actualEnvironment.Close())
}

func testNewEnvironmentConsul(t *testing.T) {
	defer resetEnvironmentFactories()
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = sallust.Default()
		v      = viper.New()

		expectedEnvironment = service.NewEnvironment()
		configuration       = strings.NewReader(`
			{
				"consul": {
					"client": {
						"address": "localhost:8500",
						"scheme": "https"
					},
					"registrations": [
						{
							"name": "test",
							"tags": ["tag1", "tag2"],
							"address": "foobar.com",
							"port": 2121
						},
						{
							"name": "test2",
							"address": "foobar.com",
							"port": 3131
						}
					],
					"watches": [
						{
							"service": "test",
							"tags": ["tag1"],
							"passingOnly": true
						},
						{
							"service": "test2",
							"passingOnly": false
						}
					]
				}
			}
		`)
	)
	v.SetConfigType("json")
	require.NoError(v.ReadConfig(configuration))

	consulEnvironmentFactory = func(l *zap.Logger, registrationScheme string, co consul.Options, eo ...service.Option) (service.Environment, error) {
		assert.Equal(logger, l)
		assert.Equal(
			consul.Options{
				Client: &api.Config{
					Address: "localhost:8500",
					Scheme:  "https",
				},
				Registrations: []api.AgentServiceRegistration{
					{
						Name:    "test",
						Tags:    []string{"tag1", "tag2"},
						Address: "foobar.com",
						Port:    2121,
					},
					{
						Name:    "test2",
						Address: "foobar.com",
						Port:    3131,
					},
				},
				Watches: []consul.Watch{
					{
						Service:     "test",
						Tags:        []string{"tag1"},
						PassingOnly: true,
					},
					{
						Service:     "test2",
						PassingOnly: false,
					},
				},
			},
			co,
		)
		return expectedEnvironment, nil
	}
	actualEnvironment, err := NewEnvironment(logger, v)
	require.NoError(err)
	require.NotNil(actualEnvironment)
	assert.Equal(expectedEnvironment, actualEnvironment)
	assert.NoError(actualEnvironment.Close())
}
func TestNewEnvironment(t *testing.T) {
	t.Run("Empty", testNewEnvironmentEmpty)
	t.Run("UnmarshalError", testNewEnvironmentUnmarshalError)
	t.Run("Fixed", testNewEnvironmentFixed)
	t.Run("Zookeeper", testNewEnvironmentZookeeper)
	t.Run("Consul", testNewEnvironmentConsul)
}
