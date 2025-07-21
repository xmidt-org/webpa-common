// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package monitor

var (
	eventCountKey string = "eventCount"
)

// EventCountKey returns the contextual logging key for the event count
func EventCountKey() string {
	return eventCountKey
}
