package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewAWSConfig(t *testing.T) {
	type AWS struct {
		Aws AWSConfig `json:"aws"`
	}
	var (
		awsCfg = AWS{Aws: AWSConfig{
			AccessKey: "accessKey",
			SecretKey: "secretKey",
			Env:       "cd",
			Sns: SNSConfig{
				Region:   "us-east-1",
				Protocol: "https",
				TopicArn: "arn:aws:sns:us-east-1:1234:test",
				UrlPath:  "/api",
			},
		}}
		assert  = assert.New(t)
		require = require.New(t)

		cfg_json, _ = json.Marshal(awsCfg)
		cfg         = bytes.NewBufferString(string(cfg_json))

		v = viper.New()
	)
	v.SetConfigType("json")
	require.Nil(v.ReadConfig(cfg))

	c, err := NewAWSConfig(v)
	require.NotNil(c)
	require.NotNil(c.Sns)
	assert.Nil(err)
	assert.Equal(awsCfg.Aws.AccessKey, c.AccessKey)
	assert.Equal(awsCfg.Aws.SecretKey, c.SecretKey)
	assert.Equal(awsCfg.Aws.Env, c.Env)
	assert.Equal(awsCfg.Aws.Sns.Protocol, c.Sns.Protocol)
	assert.Equal(awsCfg.Aws.Sns.Region, c.Sns.Region)
	assert.Equal(awsCfg.Aws.Sns.TopicArn, c.Sns.TopicArn)
	assert.Equal(awsCfg.Aws.Sns.UrlPath, c.Sns.UrlPath)
}

func TestNewAWSConfig_Invalid(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		cfg     = bytes.NewBufferString(`{}`)

		v = viper.New()
	)
	v.SetConfigType("json")
	require.Nil(v.ReadConfig(cfg))

	c, err := NewAWSConfig(v)
	assert.NotNil(err)
	assert.Nil(c)
	assert.Equal(err, fmt.Errorf("missing 'aws' key"))
}

func TestNewAWSConfig_InvalidAccessKey(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		cfg     = bytes.NewBufferString(`{
                        "aws": {
                                "accessKey": "",
                                "secretKey": "secretKey"
                }}`)

		v = viper.New()
	)
	v.SetConfigType("json")
	require.Nil(v.ReadConfig(cfg))

	c, err := NewAWSConfig(v)
	assert.NotNil(err)
	assert.Equal(fmt.Errorf("invalid AWS accesskey or secretkey"), err)
	assert.Nil(c)
}

func TestNewAWSConfig_ValidJsonConfig(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		cfg     = bytes.NewBufferString(`{
                        "aws": {
                                "accessKey": "accessKey",
                                "secretKey": "secretKey",
                                "env": "cd",
                                "sns" : {
                                        "region" : "us-east-1",
                                        "protocol" : "https",
					"topicArn" : "arn:aws:sns:us-east-1:1234:test", 
					"urlPath" : "/api"
                                }
		                }
		          }`)

		v = viper.New()
	)
	v.SetConfigType("json")
	require.Nil(v.ReadConfig(cfg))

	c, err := NewAWSConfig(v)

	assert.Nil(err)
	assert.NotNil(c)
}

func TestNewAWSConfig_InvalidSNSConfig(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		cfg     = bytes.NewBufferString(`{
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
		          }`)

		v = viper.New()
	)
	v.SetConfigType("json")
	require.Nil(v.ReadConfig(cfg))

	c, err := NewAWSConfig(v)

	assert.NotNil(err)
	assert.Nil(c)
}

func TestNewAWSConfig_ViperNil(t *testing.T) {
	assert  := assert.New(t)
	require := require.New(t)
		
	c, err := NewAWSConfig(nil)

	assert.Nil(err)
	require.NotNil(c)
	assert.Equal(c.AccessKey,"test-accessKey")
	assert.Equal(c.Sns.UrlPath,"/api/v2/aws/sns")
	
}
