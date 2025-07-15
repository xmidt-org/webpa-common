// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"github.com/c9s/goprocinfo/linux"
)

const (
	// DefaultMemoryReaderLocation is the default location for meminfo
	// under Linux
	DefaultMemoryReaderLocation string = "/proc/meminfo"
)

// MemInfoReader handles extracting the linux memory information from
// the enclosing environment.
type MemInfoReader struct {
	Location string
}

// Read parses the configured Location as if it were a linux meminfo file.
func (reader *MemInfoReader) Read() (memInfo *linux.MemInfo, err error) {
	location := reader.Location
	if len(location) == 0 {
		location = DefaultMemoryReaderLocation
	}

	return linux.ReadMemInfo(location)
}
