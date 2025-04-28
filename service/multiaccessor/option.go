// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

type option interface {
	Apply(multiHasher)
}

type options []option

func (opts options) Apply(h multiHasher) {
	for _, o := range opts {
		o.Apply(h)
	}
}

type optionFunc func(multiHasher)

func (f optionFunc) Apply(h multiHasher) {
	f(h)
}
