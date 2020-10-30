package xwebhook

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/model"
)

type pushReaderMock struct {
	mock.Mock
}

func (p *pushReaderMock) Push(item model.Item, owner string) (string, error) {
	args := p.Called(item, owner)
	return args.String(0), args.Error(1)
}

func (p *pushReaderMock) Remove(id string, owner string) (model.Item, error) {
	args := p.Called(id, owner)
	return args.Get(0).(model.Item), args.Error(1)
}

func (p *pushReaderMock) GetItems(owner string) ([]model.Item, error) {
	args := p.Called(owner)
	return args.Get(0).([]model.Item), args.Error(1)
}
