// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"testing"
)

func TestLinuxRead(t *testing.T) {
	reader := &MemInfoReader{"meminfo.test"}
	memInfo, err := reader.Read()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if memInfo == nil {
		t.Fatalf("No MemInfo returned")
	}
}

func TestNonLinunxRead(t *testing.T) {
	reader := &MemInfoReader{"nosuch"}
	memInfo, err := reader.Read()
	if err == nil {
		t.Errorf("No error returned")
	}

	if memInfo != nil {
		t.Errorf("A MemInfo should not have been returned: %v", *memInfo)
	}
}
