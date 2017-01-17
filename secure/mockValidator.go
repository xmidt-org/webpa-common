package secure

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockValidator is a stretchr mock, exposed for use by other packages
type MockValidator struct {
	mock.Mock
}

func (v *MockValidator) Validate(ctx context.Context, token *Token) (bool, error) {
	arguments := v.Called(ctx, token)
	return arguments.Bool(0), arguments.Error(1)
}
