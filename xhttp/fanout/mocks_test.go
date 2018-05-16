package fanout

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/stretchr/testify/mock"
)

type mockEndpoints struct {
	mock.Mock
}

func (m *mockEndpoints) FanoutURLs(original *http.Request) ([]*url.URL, error) {
	arguments := m.Called(original)
	first, _ := arguments.Get(0).([]*url.URL)
	return first, arguments.Error(1)
}

// generateEndpoints creates a FixedEndpoints with generated base URLs
func generateEndpoints(count int) FixedEndpoints {
	fe := make(FixedEndpoints, count)
	for i := 0; i < count; i++ {
		fe[i] = &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("host-%d.webpa.net:8080", i),
		}
	}

	return fe
}
