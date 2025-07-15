// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"encoding/json"
	"errors"
	"net"

	// nolint: typecheck
	"sync/atomic"
	"time"
)

const (
	DEFAULT_EXPIRATION_DURATION time.Duration = time.Second * 300
)

// TODO Use below to validate the input
//	https://github.com/asaskevich/govalidator

// W is the structure that represents the Webhook listener
// data we share.
//
// (Note to Wes: this follows the golang naming conventions.  webhook.Webhook "stutters",
// and this type is really the central type of this package.  Calling it a single letter is the norm.
// This could also go in the server package, in which case I'd change the name to Webhook, since
// service.Webhook works better.  See https://blog.golang.org/package-names)
type W struct {
	// Configuration for message delivery
	Config struct {
		// The URL to deliver messages to.
		URL string `json:"url"`

		// The content-type to set the messages to (unless specified by WRP).
		ContentType string `json:"content_type"`

		// The secret to use for the SHA1 HMAC.
		// Optional, set to "" to disable behavior.
		Secret string `json:"secret,omitempty"`

		// alt_urls is a list of explicit URLs that should be round robin on faliure
		AlternativeURLs []string `json:"alt_urls,omitempty"`
	} `json:"config"`

	// The URL to notify when we cut off a client due to overflow.
	// Optional, set to "" to disable behavior
	FailureURL string `json:"failure_url"`

	// The list of regular expressions to match event type against.
	Events []string `json:"events"`

	// Matcher type contains values to match against the metadata.
	Matcher struct {
		// The list of regular expressions to match device id type against.
		DeviceId []string `json:"device_id"`
	} `json:"matcher,omitempty"`

	// The specified duration for this hook to live
	Duration time.Duration `json:"duration"`

	// The absolute time when this hook is to be disabled
	Until time.Time `json:"until"`

	// The address that performed the registration
	Address string `json:"registered_from_address"`
}

func NewW(jsonString []byte, ip string) (w *W, err error) {
	w = new(W)

	err = json.Unmarshal(jsonString, w)
	if err != nil {
		var wa []W

		err = json.Unmarshal(jsonString, &wa)
		if err != nil {
			return
		}
		w = &wa[0]
	}

	err = w.sanitize(ip)
	if nil != err {
		w = nil
	}
	return
}

func (w *W) sanitize(ip string) (err error) {

	if "" == w.Config.URL {
		err = errors.New("invalid Config URL")
		return
	}

	if 0 == len(w.Events) {
		err = errors.New("invalid events")
		return
	}

	// TODO Validate content type ?  What about different types?

	if 0 == len(w.Matcher.DeviceId) {
		w.Matcher.DeviceId = []string{".*"} // match anything
	}

	if "" == w.Address && "" != ip {
		// Record the IP address the request came from
		host, _, _err := net.SplitHostPort(ip)
		if nil != _err {
			err = _err
			return
		}
		w.Address = host
	}

	// always set duration to default
	w.Duration = DEFAULT_EXPIRATION_DURATION

	if w.Until.Equal(time.Time{}) {
		w.Until = time.Now().Add(w.Duration)
	}

	return
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

func (ul *updatableList) set(list []W) {
	ul.value.Store(list)
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
	for _, newItem := range newItems {
		found := false
		var items []*W
		for i := 0; i < ul.Len(); i++ {
			items = append(items, ul.Get(i))
		}

		// we want to add items that will expire in the future
		if newItem.Until.After(time.Now()) {
			for i := 0; i < len(items) && !found; i++ {
				if items[i].ID() == newItem.ID() {
					found = true

					items[i].Matcher = newItem.Matcher
					items[i].Events = newItem.Events
					items[i].Config.ContentType = newItem.Config.ContentType
					items[i].Config.Secret = newItem.Config.Secret
					items[i].Until = newItem.Until
				}
			}

			// add item
			if !found {
				// nolint:gosec
				items = append(items, &newItem)
			}

			var itemsCopy []W
			for _, i := range items {
				itemsCopy = append(itemsCopy, *i)
			}

			// store items
			ul.set(itemsCopy)
		}
	}
}

func (ul *updatableList) Filter(filter func([]W) []W) {
	if list, ok := ul.value.Load().([]W); ok {
		copyOf := make([]W, len(list))
		copy(copyOf, list)

		ul.set(filter(copyOf))
	}
}

// NewList just creates an UpdatableList.  Don't forget:
// NewList(nil) is valid!
func NewList(initial []W) UpdatableList {
	ul := &updatableList{}
	ul.Update(initial)
	return ul
}
