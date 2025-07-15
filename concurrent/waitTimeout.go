// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package concurrent

import (
	"sync"
	"time"
)

// WaitTimeout performs a timed wait on a given sync.WaitGroup
func WaitTimeout(waitGroup *sync.WaitGroup, timeout time.Duration) bool {
	success := make(chan struct{})
	go func() {
		defer func() {
			// swallow any panics, as they'll just be from the channel
			// close if the timeout elapsed
			recover()
		}()
		defer close(success)
		waitGroup.Wait()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-success:
		return true
	case <-timer.C:
		return false
	}
}
