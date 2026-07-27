// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rehasher

import (
	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/service"
	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

const (
	RehashKeepDevice           = "rehash_keep_device"
	RehashDisconnectDevice     = "rehash_disconnect_device"
	RehashDisconnectAllCounter = "rehash_disconnect_all_count"
	RehashTimestamp            = "rehash_timestamp"
	RehashDurationMilliseconds = "rehash_duration_ms"

	ReasonLabel = "reason"

	DisconnectAllServiceDiscoveryError       = "sd_error"
	DisconnectAllServiceDiscoveryStopped     = "sd_stopped"
	DisconnectAllServiceDiscoveryNoInstances = "sd_no_instances"
)

// Metrics is the device module function that adds default device metrics
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name: RehashKeepDevice,
			// nolint: goconst
			Type:       "gauge",
			LabelNames: []string{service.ServiceLabel},
		},
		{
			Name: RehashDisconnectDevice,
			// nolint: goconst
			Type:       "gauge",
			LabelNames: []string{service.ServiceLabel},
		},
		{
			Name: RehashDisconnectAllCounter,
			// nolint: goconst
			Type:       "counter",
			LabelNames: []string{service.ServiceLabel, ReasonLabel},
		},
		{
			Name: RehashTimestamp,
			// nolint: goconst
			Type:       "gauge",
			LabelNames: []string{service.ServiceLabel},
		},
		{
			Name: RehashDurationMilliseconds,
			// nolint: goconst
			Type:       "gauge",
			LabelNames: []string{service.ServiceLabel},
		},
	}
}
