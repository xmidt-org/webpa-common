package client

import (
	"errors"
	"time"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
)

func Initialize(v *viper.Viper, r xmetrics.Registry, l log.Logger, or OutboundMetricOptions, sr xhttp.ShouldRetryFunc, sl func(time.Duration)) (*WebPAClient, error) {
	clientConfig, err := viperToHTTPClientConfig(v)
	if err != nil {
		return nil, err
	}

	om := NewOutboundMeasures(r)
	client := DecorateClientWithMetrics(or, om, clientConfig.NewClient())

	ok := clientConfig.RetryOptionsConfig.IsEmpty()
	switch ok {
	case true:
		retryOptions := xhttp.RetryOptions{
			Logger:      l,
			Interval:    clientConfig.RetryOptionsConfig.interval(),
			Retries:     clientConfig.RetryOptionsConfig.retries(),
			Sleep:       sl,
			ShouldRetry: sr,
			Counter:     om.Retries,
		}

		return NewWebPAClient(om, xhttp.RetryTransactor(retryOptions, client.Do)), nil
	case false:
		return NewWebPAClient(om, client.Do), nil
	default:
		return nil, errors.New("Failed to initialize webPAClient")
	}
}
