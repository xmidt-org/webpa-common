// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import "net/http"

// NilConstructor is an Alice-style decorator for http.Handler instances that does no decoration,
// i.e. it simply returns its next handler unmodified.  This is useful in cases where returning nil
// from configuration is undesirable.  Configuration code can always return a non-nil constructor,
// using this function in cases where no decoration has been configured.
func NilConstructor(next http.Handler) http.Handler {
	return next
}
