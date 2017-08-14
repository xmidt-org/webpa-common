package wrp

import (
	"github.com/stretchr/testify/mock"
	"github.com/ugorji/go/codec"
)

type mockEncodeListener struct {
	mock.Mock
}

func (m *mockEncodeListener) BeforeEncode() error {
	return m.Called().Error(0)
}

func (m *mockEncodeListener) CodecEncodeSelf(e *codec.Encoder) {
	m.Called(e)
}

func (m *mockEncodeListener) CodecDecodeSelf(e *codec.Decoder) {
	m.Called(e)
}
