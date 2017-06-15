package aws

import (
	"fmt"
	"github.com/spf13/viper"
)

const (
	// AWSKey is the subkey used to load AWS configuration (e.g. AWSConfig)
	AWSKey = "aws"
)

// NewAWSConfig produces AWSConfig from Viper environment
func NewAWSConfig(v *viper.Viper) (c *AWSConfig, err error) {
	c = new(AWSConfig)
	if v != nil && v.Sub(AWSKey) != nil {
		v = v.Sub(AWSKey)
		err = v.Unmarshal(c)
	} else if v != nil && v.Sub(AWSKey) == nil {
		return nil, fmt.Errorf("missing 'aws' key")
	} else {
		// If viper is nil then for test purposes initialize default AWSConfig object
		c.AccessKey = "test-accessKey"
		c.SecretKey = "test-secretKey"
		c.Env = "test"
		c.Sns = SNSConfig{
			Region:   "us-east-1",
			Protocol: "http",
			TopicArn: "arn:aws:sns:us-east-1:1234:test-topic",
			UrlPath:  "/api/v2/aws/sns",
		}
	}

	if nil != err {
		return nil, err
	}

	if "" == c.AccessKey || c.SecretKey == "" {
		return nil, fmt.Errorf("invalid AWS accesskey or secretkey")
	}

	if c.Sns.Region == "" || c.Sns.TopicArn == "" || c.Sns.UrlPath == "" {
		return nil, fmt.Errorf("invalid sns config %#v", c.Sns)
	}

	return
}
