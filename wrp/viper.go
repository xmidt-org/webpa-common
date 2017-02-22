package wrp

import (
	"github.com/spf13/viper"
)

const (
	// ViperKey is the usual subkey used to load WRP configuration (e.g. PoolFactory)
	ViperKey = "wrp"
)

// PoolFactory is a configurable Factory for pooled WRP encoders and decoders.
type PoolFactory struct {
	DecoderPoolSize   int
	EncoderPoolSize   int
	InitialBufferSize int
}

func NewPoolFactory(v *viper.Viper) (pf *PoolFactory, err error) {
	pf = new(PoolFactory)
	if v != nil {
		err = v.Unmarshal(pf)
	}

	return
}

func (pf *PoolFactory) NewEncoderPool(f Format) *EncoderPool {
	return NewEncoderPool(pf.EncoderPoolSize, pf.InitialBufferSize, f)
}

func (pf *PoolFactory) NewDecoderPool(f Format) *DecoderPool {
	return NewDecoderPool(pf.DecoderPoolSize, f)
}
