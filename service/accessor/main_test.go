// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package accessor

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	vnodeCounts   []int
	addressCounts []int
)

func TestMain(m *testing.M) {
	var (
		vnodeCountsValue   string
		addressCountsValue string
	)

	flag.StringVar(&vnodeCountsValue, "vnodeCounts", "100,200,300,400,500", "list of vnode counts to benchmark")
	flag.StringVar(&addressCountsValue, "addressCounts", "1,5,10,50", "list of address counts to benchmark")
	flag.Parse()

	for _, v := range strings.Split(vnodeCountsValue, ",") {
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		vnodeCounts = append(vnodeCounts, i)
	}

	for _, v := range strings.Split(addressCountsValue, ",") {
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		addressCounts = append(addressCounts, i)
	}

	sort.Ints(vnodeCounts)
	sort.Ints(addressCounts)
	os.Exit(m.Run())
}
