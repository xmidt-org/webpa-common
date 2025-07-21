// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/stretchr/testify/mock"
)

type MockSVC struct {
	snsiface.SNSAPI
	mock.Mock
}

type MockValidator struct {
	mock.Mock
}

func (m *MockSVC) Subscribe(input *sns.SubscribeInput) (*sns.SubscribeOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.SubscribeOutput), args.Error(1)
}

func (m *MockSVC) ConfirmSubscription(input *sns.ConfirmSubscriptionInput) (*sns.ConfirmSubscriptionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.ConfirmSubscriptionOutput), args.Error(1)
}

func (m *MockSVC) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.PublishOutput), args.Error(1)
}

func (m *MockSVC) Unsubscribe(input *sns.UnsubscribeInput) (*sns.UnsubscribeOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.UnsubscribeOutput), args.Error(1)
}

func (m *MockSVC) ListSubscriptionsByTopic(input *sns.ListSubscriptionsByTopicInput) (*sns.ListSubscriptionsByTopicOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sns.ListSubscriptionsByTopicOutput), args.Error(1)
}

func (m *MockValidator) Validate(msg *SNSMessage) (bool, error) {
	args := m.Called(msg)
	return args.Get(0).(bool), args.Error(1)
}
