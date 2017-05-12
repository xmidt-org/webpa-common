package aws

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
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
