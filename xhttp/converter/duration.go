// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package converter

import (
	"reflect"
	"time"
)

func Duration(v string) reflect.Value {
	d, err := time.ParseDuration(v)
	if err != nil {
		return reflect.Value{}
	}

	return reflect.ValueOf(d)
}
