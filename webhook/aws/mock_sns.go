package aws

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/stretchr/testify/mock"
)

type mockSVC struct {
	snsiface.SNSAPI
    mock.Mock
}

func (m *mockSVC) Subscribe( input *sns.SubscribeInput) (*sns.SubscribeOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.SubscribeOutput), args.Error(1)
}

func (m *mockSVC) ConfirmSubscription(input *sns.ConfirmSubscriptionInput) (*sns.ConfirmSubscriptionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.ConfirmSubscriptionOutput), args.Error(1)
}

func (m *mockSVC) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.PublishOutput), args.Error(1)
}

func (m *mockSVC) Unsubscribe(input *sns.UnsubscribeInput) (*sns.UnsubscribeOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.UnsubscribeOutput), args.Error(1)
}