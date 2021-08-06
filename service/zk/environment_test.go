package zk

import (
	"errors"
	"testing"

	"github.com/go-kit/kit/log"
	gokitzk "github.com/go-kit/kit/sd/zk"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/webpa-common/v2/service"
)

func testNewEnvironmentEmpty(t *testing.T) {
	defer resetClientFactory()

	var (
		assert        = assert.New(t)
		clientFactory = prepareMockClientFactory()
	)

	e, err := NewEnvironment(nil, Options{})
	assert.Nil(e)
	assert.Equal(service.ErrIncomplete, err)

	clientFactory.AssertExpectations(t)
}

func testNewEnvironmentClientError(t *testing.T) {
	defer resetClientFactory()

	var (
		assert = assert.New(t)

		clientFactory       = prepareMockClientFactory()
		expectedClientError = errors.New("expected client error")

		zo = Options{
			Client: Client{
				Connection: "www.shinola.net:383",
			},
			Watches: []string{"/some/where"},
		}
	)

	clientFactory.On("NewClient",
		[]string{"www.shinola.net:383"},
		mock.MatchedBy(func(l log.Logger) bool { return l != nil }),
		mock.MatchedBy(func(o []gokitzk.Option) bool { return len(o) == 2 }),
	).Return(nil, expectedClientError).Once()

	e, actualClientError := NewEnvironment(nil, zo)
	assert.Nil(e)
	assert.Equal(expectedClientError, actualClientError)

	clientFactory.AssertExpectations(t)
}

func testNewEnvironmentInstancerError(t *testing.T) {
	defer resetClientFactory()

	var (
		assert = assert.New(t)

		clientFactory          = prepareMockClientFactory()
		expectedInstancerError = errors.New("expected instancer error")
		client                 = new(mockClient)
		zkEvents               = make(chan zk.Event, 5)

		zo = Options{
			Client: Client{
				Connection: "sherbert.com:9999",
			},
			Watches: []string{"/good", "/bad"},
		}
	)

	clientFactory.On("NewClient",
		[]string{"sherbert.com:9999"},
		mock.MatchedBy(func(l log.Logger) bool { return l != nil }),
		mock.MatchedBy(func(o []gokitzk.Option) bool { return len(o) == 2 }),
	).Return(client, error(nil)).Once()

	client.On("CreateParentNodes", "/good").Return(error(nil)).Once()
	client.On("GetEntries", "/good").Return([]string{"good1", "good2"}, (<-chan zk.Event)(zkEvents), error(nil)).Once()
	client.On("CreateParentNodes", "/bad").Return(expectedInstancerError).Once()

	client.On("Stop").Once()

	e, actualInstancerError := NewEnvironment(nil, zo)
	assert.Nil(e)
	assert.Equal(expectedInstancerError, actualInstancerError)

	clientFactory.AssertExpectations(t)
	client.AssertExpectations(t)
}

func testNewEnvironmentFull(t *testing.T) {
	defer resetClientFactory()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger        = logging.NewTestLogger(nil, t)
		clientFactory = prepareMockClientFactory()
		client        = new(mockClient)
		zkEvents      = make(chan zk.Event, 5)

		zo = Options{
			Client: Client{
				Connection: "someserver.net:7171",
			},
			Registrations: []Registration{
				Registration{
					Name:    "foobar",
					Path:    "/test1",
					Address: "foobar.net",
					Port:    1717,
					Scheme:  "https",
				},
				Registration{
					Name:    "foobar",
					Path:    "/test1",
					Address: "foobar.net",
					Port:    1717,
					Scheme:  "https",
				}, // duplicate should be ignored
			},
			Watches: []string{"/test1", "/test2", "/test2"}, // duplicate should be ignored
		}
	)

	clientFactory.On("NewClient",
		[]string{"someserver.net:7171"},
		logger,
		mock.MatchedBy(func(o []gokitzk.Option) bool { return len(o) == 2 }),
	).Return(client, error(nil)).Once()

	client.On("CreateParentNodes", "/test1").Return(error(nil)).Once()
	client.On("GetEntries", "/test1").Return([]string{"instance1"}, (<-chan zk.Event)(zkEvents), error(nil)).Once()
	client.On("CreateParentNodes", "/test2").Return(error(nil)).Once()
	client.On("GetEntries", "/test2").Return([]string{"instance2"}, (<-chan zk.Event)(zkEvents), error(nil)).Once()

	client.On("Register",
		mock.MatchedBy(func(s *gokitzk.Service) bool {
			return s.Path == "/test1" && s.Name == "foobar" && string(s.Data) == "https://foobar.net:1717"
		}),
	).Return(error(nil)).Once()

	client.On("Deregister",
		mock.MatchedBy(func(s *gokitzk.Service) bool {
			return s.Path == "/test1" && s.Name == "foobar" && string(s.Data) == "https://foobar.net:1717"
		}),
	).Return(error(nil)).Twice()

	client.On("Stop").Once()

	e, err := NewEnvironment(logger, zo)
	require.NoError(err)
	require.NotNil(e)

	e.Register()
	e.Deregister()

	assert.NoError(e.Close())

	clientFactory.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestNewEnvironment(t *testing.T) {
	t.Run("Empty", testNewEnvironmentEmpty)
	t.Run("ClientError", testNewEnvironmentClientError)
	t.Run("InstancerError", testNewEnvironmentInstancerError)
	t.Run("Full", testNewEnvironmentFull)
}
