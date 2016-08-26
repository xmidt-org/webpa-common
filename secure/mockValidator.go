package secure

import (
	"github.com/stretchr/testify/mock"
)

// MockValidator is a stretchr mock, exposed for use by other packages
type MockValidator struct {
	mock.Mock
}

func (v *MockValidator) Validate(token *Token) (bool, error) {
	arguments := v.Called(token)
	return arguments.Bool(0), arguments.Error(1)
}
