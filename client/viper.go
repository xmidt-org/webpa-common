package client

import (
	"github.com/spf13/viper"
)

func viperToHTTPClientConfig(v *viper.Viper) (*HTTPClientConfig, error) {
	clientConfig := new(HTTPClientConfig)
	v.Unmarshal(clientConfig)

	return clientConfig, nil
}
