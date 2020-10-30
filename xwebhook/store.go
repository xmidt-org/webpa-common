package xwebhook

import (
	"github.com/xmidt-org/argus/model"
)

type PushReader interface {
	Pusher
	Reader
}

type Pusher interface {
	// Push applies user configurable for registering an item returning the id
	// i.e. updated the storage with said item.
	Push(item model.Item, owner string) (string, error)

	// Remove will remove the item from the store
	Remove(id string, owner string) (model.Item, error)
}

type Reader interface {
	// GeItems will return all the current items or an error.
	GetItems(owner string) ([]model.Item, error)
}
