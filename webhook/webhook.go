package webhook

import (
	"sync/atomic"
)

// W is the structure that represents the Webhook listener
// data we share.
//
// (Note to Wes: this follows the golang naming conventions.  webhook.Webhook "stutters",
// and this type is really the central type of this package.  Calling it a single letter is the norm.
// This could also go in the server package, in which case I'd change the name to Webhook, since
// service.Webhook works better.  See https://blog.golang.org/package-names)
type W struct {
	Config struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
		Secret      string `json:"secret"`
	} `json:"config"`
	Matcher struct {
		DeviceId []string `json:"device_id"`
	} `json:"matcher"`
	Events   []string `json:"events"`
	Groups   []string `json:"groups"`
	Duration int64    `json:"duration"`
	Until    int64    `json:"until"`
	Address  string   `json:"address"`
}

// ID creates the canonical string identifing a WebhookListener
func (w *W) ID() string {
	return w.Config.URL
}

// List is a read-only random access interface to a set of W's
// We don't necessarily need an implementation of just this interface alone.
type List interface {
	Len() int
	Get(int) *W
}

// UpdatableList is mutable list that can be updated en masse
type UpdatableList interface {
	List

	// Update performs a bulk update of this webhooks known to this list
	Update([]W)

	// Filter atomically filters the elements of this list
	Filter(func([]W) []W)
}

type updatableList struct {
	value atomic.Value
}

func (ul *updatableList) Len() int {
	if list, ok := ul.value.Load().([]W); ok {
		return len(list)
	}

	return 0
}

func (ul *updatableList) Get(index int) *W {
	if list, ok := ul.value.Load().([]W); ok {
		return &list[index]
	}

	// TODO: design choice.  may want to panic here, to mimic
	// the behavior of the golang runtime for slices.  Alternatively,
	// could return a second parameter that is an error (consistentHash does that).
	return nil
}

func (ul *updatableList) Update(newItems []W) {
	ul.value.Store(newItems)
}

func (ul *updatableList) Filter(filter func([]W) []W) {
	if list, ok := ul.value.Load().([]W); ok {
		copyOf := make([]W, len(list))
		for i, w := range list {
			copyOf[i] = w
		}

		ul.Update(filter(copyOf))
	}
}

// NewList just creates an UpdatableList.  Don't forget:
// NewList(nil) is valid!
func NewList(initial []W) UpdatableList {
	ul := &updatableList{}
	ul.Update(initial)
	return ul
}
