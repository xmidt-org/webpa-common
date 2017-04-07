package aws

import (
	"github.com/spf13/viper"
	"fmt"
)

const (
	// AWSKey is the subkey used to load AWS configuration (e.g. AWSConfig)
	AWSKey = "aws"
)

// NewAWSConfig produces AWSConfig from Viper environment
func NewAWSConfig(v *viper.Viper) (c *AWSConfig, err error) {
	c = new(AWSConfig)
	if v != nil {
		err = v.Unmarshal(c)
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
