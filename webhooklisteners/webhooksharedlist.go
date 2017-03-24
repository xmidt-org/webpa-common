package webhooklisteners

import (
//"time"
)

// WebHookSharedListFactory provides the input values to create a
// WebHookSharedList service object with New()
type WebHookSharedListFactory struct {
}

// New creates a new WebHookSharedList service object
func (w *WebHookSharedListFactory) New() (out *WebHookSharedList, err error) {
	// TODO do the work
	return
}

// WebHookSharedList is the interface that provides the shared list of webhooks
// across the system
type WebHookSharedList struct {
	// TODO add parameters needed to operate the service
}

// Add adds a WebHookListener to the shared slist
func (w *WebHookSharedList) Add(item WebHookListener) (err error) {
	// TODO the work of sending this to AWS
	return
}

// Listen adds a listener function to be notifed when there is a change in the
// list.  The entire list is given to the listener for processing.
func (w *WebHookSharedList) Listen(func([]WebHookListener)) (err error) {
	// TODO the work of getting this data from AWS
	return
}
