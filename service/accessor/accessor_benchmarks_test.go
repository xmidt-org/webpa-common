// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package accessor

import (
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

const addressSeed int64 = 2387240836734099856

func generateAddresses(random io.Reader, count, length int) []string {
	var (
		buffer    = make([]byte, length)
		addresses []string
	)

	for i := 0; i < count; i++ {
		random.Read(buffer)
		addresses = append(
			addresses,
			base64.RawURLEncoding.EncodeToString(buffer),
		)
	}

	return addresses
}

func BenchmarkAccessor(b *testing.B) {
	b.Log("-vnodeCounts set to", vnodeCounts)
	b.Log("-addressCounts set to", addressCounts)

	var (
		random    = rand.New(rand.NewSource(addressSeed))
		addresses = generateAddresses(random, addressCounts[len(addressCounts)-1], 32)
	)

	b.Run("Create", func(b *testing.B) {
		for _, vnodeCount := range vnodeCounts {
			for _, addressCount := range addressCounts {
				var (
					name          = fmt.Sprintf("(vnodeCount=%d,addressCount=%d)", vnodeCount, addressCount)
					testAddresses = addresses[:addressCount]
				)

				b.Run(name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						NewConsistentAccessorFactory(vnodeCount)(testAddresses)
					}
				})
			}
		}
	})

	b.Run("Get", func(b *testing.B) {
		for _, vnodeCount := range vnodeCounts {
			for _, addressCount := range addressCounts {
				var (
					name     = fmt.Sprintf("(vnodeCount=%d,addressCount=%d)", vnodeCount, addressCount)
					accessor = NewConsistentAccessorFactory(vnodeCount)(addresses[:addressCount])
					key      = make([]byte, 32)
				)

				random.Read(key)
				b.Run(name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						accessor.Get(key)
					}
				})
			}
		}
	})
}
