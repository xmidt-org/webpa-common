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

func doOldHookConvert(oldHook oldW) (newHook W) {
	newHook.Config.URL = oldHook.Config.URL
	newHook.Config.ContentType = oldHook.Config.ContentType
	newHook.Config.Secret = oldHook.Config.Secret
	newHook.Events = oldHook.Events
	newHook.Matcher = oldHook.Matcher
	newHook.Duration = time.Duration(oldHook.Duration) * time.Second

	newHook.Until = time.Time{}
	if oldHook.Until > 0 {
		newHook.Until = time.Unix(oldHook.Until, 0)
	}

	newHook.Address = oldHook.Address
	
	return
}

func convertOldHooksToNewHooks(body []byte) (hooks []W, err error) {
	var oldHooks []oldW
	err = json.Unmarshal(body, &oldHooks)
	if err != nil {
		return
	}

	for _, oldHook := range oldHooks {
		hooks = append(hooks, doOldHookConvert(oldHook))
	}

	return
}
