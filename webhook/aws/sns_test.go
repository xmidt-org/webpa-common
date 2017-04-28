package aws

import (
	"testing"
	"bytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_AWS_CONFIG = `{
	"aws": {
        "accessKey": "test-accessKey",
        "secretKey": "test-secretKey",
        "env": "test",
        "sns" : {
	        "region" : "us-east-1",
            "protocol" : "http",
			"topicArn" : "arn:aws:sns:us-east-1:1234:test-topic", 
			"urlPath" : "/api/v2/aws/sns"
    } } }`
	TEST_AWS_CFG_ERR = `{
                        "aws": {
                                "accessKey": "accessKey",
                                "secretKey": "secretKey",
                                "env": "cd",
                                "sns" : {
                                        "region" : "us-east-1",
                                        "protocol" : "https", 
                                        "urlPath" : "/api"
                                }
		                }
		          }`
)

func SetUpTestViperInstance(config string) *viper.Viper {
	
	cfg := bytes.NewBufferString(config)
	v := viper.New()
	v.SetConfigType("json")
	v.ReadConfig(cfg)
	return v
}

func TestNewSNSServerSuccess(t *testing.T) {
	assert  := assert.New(t)
	require := require.New(t)
	
	v := SetUpTestViperInstance(TEST_AWS_CONFIG)
	ss, err := NewSNSServer(v)
	
	require.NotNil(ss)
	require.Nil(err)
	assert.NotNil(ss.Config)
	assert.NotNil(ss.SVC)
	assert.Equal("test",ss.Config.Env)
	assert.Equal("us-east-1", ss.Config.Sns.Region)
}

func TestNewSNSServerAWSConfigError(t *testing.T) {
	assert  := assert.New(t)
	require := require.New(t)
	
	v := SetUpTestViperInstance(TEST_AWS_CFG_ERR)
	ss, err := NewSNSServer(v)
	
	require.NotNil(err)
	assert.Nil(ss)
}
