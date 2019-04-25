package client

import (
	"errors"
	"time"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
)

func Initialize(v *viper.Viper, r xmetrics.Registry, l log.Logger, sr xhttp.ShouldRetryFunc, sl func(time.Duration)) (*WebPAClient, error) {
	clientConfig, err := viperToHTTPClientConfig(v)
	if err != nil {
		return nil, err
	}

	om := NewOutboundMeasures(r)

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

		transactor, err := clientConfig.NewTransactor()
		if err != nil {
			return nil, err
		}

		transactor = xhttp.RetryTransactor(retryOptions, transactor)

		return NewWebPAClient(om, transactor), nil
	case false:
		client, err := clientConfig.NewClient()
		if err != nil {
			return nil, err
		}

		return NewWebPAClient(om, client.Do), nil
	default:
		return nil, errors.New("Failed to initialize webPAClient")
	}
}
