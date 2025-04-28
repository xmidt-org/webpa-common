// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

var (
	defaultHasherType = HostnameType
)

func Instances(s []string) option {
	return optionFunc(func(hs multiHasher) {
		for _, hs := range hs {
			for _, i := range s {
				hs.Add(i)
			}
		}
	})
}

func VnodeCount(c int) option {
	return optionFunc(func(hs multiHasher) {
		for _, hs := range hs {
			hs.SetVnodeCount(c)
		}
	})
}

func Hasher(hts ...HasherType) option {
	return optionFunc(func(hs multiHasher) {
		for _, ht := range hts {
			switch ht {
			case RawURLType:
				hs[RawURLType] = NewHasher(RawURLNormalizer)
			case HostnameType:
				hs[HostnameType] = NewHasher(HostnameNormalizer)
			}
		}
	})
}
