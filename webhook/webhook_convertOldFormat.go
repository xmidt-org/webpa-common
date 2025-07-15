// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"encoding/json"
	"time"
)

type oldW struct {
	// Configuration for message delivery
	Config struct {
		// The URL to deliver messages to.
		URL string `json:"url"`

		// The content-type to set the messages to (unless specified by WRP).
		ContentType string `json:"content_type"`

		// The secret to use for the SHA1 HMAC.
		// Optional, set to "" to disable behavior.
		Secret string `json:"secret,omitempty"`
	} `json:"config"`

	// The list of regular expressions to match event type against.
	Events []string `json:"events"`

	// Matcher type contains values to match against the metadata.
	Matcher struct {
		// The list of regular expressions to match device id type against.
		DeviceId []string `json:"device_id"`
	} `json:"matcher,omitempty"`

	// The specified duration for this hook to live
	Duration int64 `json:"duration"`

	// The absolute time when this hook is to be disabled
	Until int64 `json:"until"`

	// The address that performed the registration
	Address string `json:"registered_from_address"`
}

func doOldHookConvert(jsonString []byte) (w *W, err error) {
	old := new(oldW)
	err = json.Unmarshal(jsonString, old)
	if err != nil {
		return
	}

	return oldToNewHookConversion(old)
}

func oldToNewHookConversion(old *oldW) (w *W, err error) {
	w = new(W)
	w.Config.URL = old.Config.URL
	w.Config.ContentType = old.Config.ContentType
	w.Config.Secret = old.Config.Secret
	w.Events = old.Events
	w.Matcher = old.Matcher
	w.Address = old.Address

	if old.Duration <= 0 || old.Duration > 300000 {
		old.Duration = 300000
	}
	w.Duration = time.Duration(old.Duration) * time.Second

	w.Until = time.Time{}
	if old.Until > 0 {
		w.Until = time.Unix(old.Until, 0)
	}

	err = w.sanitize("")
	if nil != err {
		w = nil
	}

	return
}

func convertOldHooksToNewHooks(body []byte) (hooks []W, err error) {
	var oldHooks []oldW
	err = json.Unmarshal(body, &oldHooks)
	if err != nil {
		return
	}

	for _, oldHook := range oldHooks {
		var old *W
		// nolint gosec
		old, err = oldToNewHookConversion(&oldHook)
		if nil != err {
			hooks = nil
			return
		}
		hooks = append(hooks, *old)
	}

	return
}
