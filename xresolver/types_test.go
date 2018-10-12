package xresolver

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"strconv"
	"testing"
)

func testRoute(ip string) Route {
	return Route{
		Host:   ip,
		Scheme: "http",
	}
}

func TestRoundRobinOperations(t *testing.T) {
	assert := assert.New(t)

	balancer := NewRoundRobinBalancer()

	expected := testRoute("127.0.0.1")

	records, err := balancer.Get()
	assert.Error(err)
	assert.Empty(records)

	err = balancer.Remove(expected)
	assert.Error(err)

	err = balancer.Add(expected)
	assert.NoError(err)

	err = balancer.Add(expected)
	assert.Error(err)

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal(1, len(records))
	assert.Equal(expected, records[0])

	err = balancer.Remove(expected)
	assert.NoError(err)

	records, err = balancer.Get()
	assert.Error(err)
	assert.Empty(records)

	balancer.Update([]Route{
		expected,
		testRoute("127.0.0.1"),
		testRoute("8.8.8.8"),
		testRoute("1.1.1.1"),
		testRoute("127.0.0.1"),
	})

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal(3, len(records))
	assert.Equal([]Route{
		expected,
		testRoute("8.8.8.8"),
		testRoute("1.1.1.1"),
	}, records)
}

func TestRoundRobinOrder(t *testing.T) {
	assert := assert.New(t)

	balancer := NewRoundRobinBalancer()

	localAddress := testRoute("127.0.0.1")
	googleAddres := testRoute("8.8.8.8")
	addressOnes := testRoute("1.1.1.1")

	balancer.Add(localAddress)
	balancer.Add(googleAddres)
	balancer.Add(addressOnes)

	records, err := balancer.Get()
	assert.NoError(err)
	assert.Equal([]Route{localAddress, googleAddres, addressOnes}, records, "records are assumed to be the order in which added")

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal([]Route{googleAddres, addressOnes, localAddress}, records, "records should rotate on another Get()")

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal([]Route{addressOnes, localAddress, googleAddres}, records, "records should rotate on another Get()")

	err = balancer.Remove(googleAddres)
	assert.NoError(err)

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal([]Route{localAddress, addressOnes}, records, "records should rotate on another Get()")
}

/**
Processor Speed: 2.8 GHz
Number of Processors: 1
Total Number of Cores: 4
L2 Cache (per Core): 256 KB
L3 Cache: 6 MB
Memory: 16 GB
Version: OS X 10.13.5

BenchmarkRoundRobinAdd/add/1-8         	10000000	       237 ns/op	      40 B/op	       3 allocs/op
BenchmarkRoundRobinAdd/add/2-8         	 3000000	       440 ns/op	      80 B/op	       6 allocs/op
BenchmarkRoundRobinAdd/add/4-8         	 2000000	       880 ns/op	     160 B/op	      12 allocs/op
BenchmarkRoundRobinAdd/add/8-8         	 1000000	      1782 ns/op	     320 B/op	      24 allocs/op
BenchmarkRoundRobinAdd/add/16-8        	  300000	      3628 ns/op	     640 B/op	      48 allocs/op
BenchmarkRoundRobinAdd/add/32-8        	  200000	      7217 ns/op	    1280 B/op	      96 allocs/op
BenchmarkRoundRobinAdd/add/64-8        	  100000	     14479 ns/op	    2560 B/op	     192 allocs/op
BenchmarkRoundRobinAdd/add/128-8       	   50000	     30517 ns/op	    5344 B/op	     412 allocs/op
BenchmarkRoundRobinAdd/add/256-8       	   20000	     62290 ns/op	   11490 B/op	     924 allocs/op
BenchmarkRoundRobinAdd/add/512-8       	   10000	    121836 ns/op	   22984 B/op	    1848 allocs/op
BenchmarkRoundRobinAdd/add/1024-8      	    5000	    248794 ns/op	   45986 B/op	    3696 allocs/op
*/

func BenchmarkRoundRobinAdd(b *testing.B) {
	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("add/%d", n), func(b *testing.B) {
			balancer := NewRoundRobinBalancer()
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				for index := 0; index < n; index++ {
					balancer.Add(testRoute(IPv4Address(int64(index))))
				}
			}
		})
	}
}

/**
Processor Speed: 2.8 GHz
Number of Processors: 1
Total Number of Cores: 4
L2 Cache (per Core): 256 KB
L3 Cache: 6 MB
Memory: 16 GB
Version: OS X 10.13.5

BenchmarkRoundRobinRemove/remove/1-8   	 1000000	      1206 ns/op	      48 B/op	       3 allocs/op
BenchmarkRoundRobinRemove/remove/2-8   	 1000000	      1685 ns/op	      96 B/op	       6 allocs/op
BenchmarkRoundRobinRemove/remove/4-8   	  500000	      2411 ns/op	     192 B/op	      12 allocs/op
BenchmarkRoundRobinRemove/remove/8-8   	  300000	      4299 ns/op	     384 B/op	      24 allocs/op
BenchmarkRoundRobinRemove/remove/16-8  	  200000	      9864 ns/op	     768 B/op	      48 allocs/op
BenchmarkRoundRobinRemove/remove/32-8  	   50000	     24964 ns/op	    1536 B/op	      96 allocs/op
BenchmarkRoundRobinRemove/remove/64-8  	   20000	     66353 ns/op	    3072 B/op	     192 allocs/op
BenchmarkRoundRobinRemove/remove/128-8 	   10000	    213464 ns/op	    6144 B/op	     384 allocs/op
BenchmarkRoundRobinRemove/remove/256-8 	    2000	    737753 ns/op	   12288 B/op	     768 allocs/op
BenchmarkRoundRobinRemove/remove/512-8 	     500	   2773625 ns/op	   24576 B/op	    1536 allocs/op
BenchmarkRoundRobinRemove/remove/1024-8      100	  10768748 ns/op	   49152 B/op	    3072 allocs/op

*/

func BenchmarkRoundRobinRemove(b *testing.B) {
	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("remove/%d", n), func(b *testing.B) {
			b.ReportAllocs()
			balancer := NewRoundRobinBalancer()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				records := make([]string, n)
				for index := 0; index < n; index++ {
					ip := IPv4Address(int64(index))
					records[index] = ip
					balancer.Add(testRoute(ip))
				}
				b.StartTimer()
				for index := 0; index < n; index++ {
					err := balancer.Remove(testRoute(records[index]))
					if err != nil {
						panic(err)
					}
				}
			}
		})
	}
}

/**
Processor Speed: 2.8 GHz
Number of Processors: 1
Total Number of Cores: 4
L2 Cache (per Core): 256 KB
L3 Cache: 6 MB
Memory: 16 GB
Version: OS X 10.13.5

BenchmarkRoundRobinUpdate/update/1-8             1000000              1460 ns/op             336 B/op          5 allocs/op
BenchmarkRoundRobinUpdate/update/2-8             1000000              1780 ns/op             416 B/op          8 allocs/op
BenchmarkRoundRobinUpdate/update/4-8             1000000              2268 ns/op             576 B/op         14 allocs/op
BenchmarkRoundRobinUpdate/update/8-8              500000              3352 ns/op             896 B/op         26 allocs/op
BenchmarkRoundRobinUpdate/update/16-8             200000              7246 ns/op            2909 B/op         52 allocs/op
BenchmarkRoundRobinUpdate/update/32-8             100000             11500 ns/op            6148 B/op        102 allocs/op
BenchmarkRoundRobinUpdate/update/64-8             100000             22976 ns/op           13140 B/op        201 allocs/op
BenchmarkRoundRobinUpdate/update/128-8             30000             43134 ns/op           26565 B/op        395 allocs/op
BenchmarkRoundRobinUpdate/update/256-8             20000             82270 ns/op           51227 B/op        780 allocs/op
BenchmarkRoundRobinUpdate/update/512-8             10000            166695 ns/op          102301 B/op       1558 allocs/op
BenchmarkRoundRobinUpdate/update/1024-8             5000            332736 ns/op          204300 B/op       3113 allocs/op
*/

func BenchmarkRoundRobinUpdate(b *testing.B) {
	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("update/%d", n), func(b *testing.B) {
			b.ReportAllocs()
			balancer := NewRoundRobinBalancer()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				routes := make([]Route, n)
				for index := 0; index < n; index++ {
					routes[index] = testRoute(IPv4Address(int64(index)))
				}
				b.StartTimer()

				balancer.Update(routes)
			}
		})
	}
}

/**
Processor Speed: 2.8 GHz
Number of Processors: 1
Total Number of Cores: 4
L2 Cache (per Core): 256 KB
L3 Cache: 6 MB
Memory: 16 GB
Version: OS X 10.13.5

BenchmarkRoundRobinGet/get/1-8                  	10000000	       231 ns/op	      48 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/2-8                  	 5000000	       282 ns/op	      80 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/4-8                  	 5000000	       349 ns/op	     160 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/8-8                  	 3000000	       436 ns/op	     320 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/16-8                 	 2000000	       786 ns/op	     640 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/32-8                 	 1000000	      1402 ns/op	    1280 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/64-8                 	  500000	      2615 ns/op	    2688 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/128-8                	  300000	      5121 ns/op	    5376 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/256-8                	  200000	      9817 ns/op	   10240 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/512-8                	  100000	     19733 ns/op	   20480 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/1024-8               	   50000	     44176 ns/op	   40960 B/op	       1 allocs/op
*/

func BenchmarkRoundRobinGet(b *testing.B) {
	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("get/%d", n), func(b *testing.B) {
			b.ReportAllocs()

			balancer := NewRoundRobinBalancer()
			for index := 0; index < n; index++ {
				balancer.Add(testRoute(IPv4Address(int64(index))))
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				records, _ := balancer.Get()
				if len(records) == 0 {
					b.Fatal(errors.New("no records"))
				}
			}
		})
	}
}

func IPv4Address(ipInt int64) string {
	// need to do two bit shifting and “0xff” masking
	b0 := strconv.FormatInt((ipInt>>24)&0xff, 10)
	b1 := strconv.FormatInt((ipInt>>16)&0xff, 10)
	b2 := strconv.FormatInt((ipInt>>8)&0xff, 10)
	b3 := strconv.FormatInt((ipInt & 0xff), 10)

	return b0 + "." + b1 + "." + b2 + "." + b3
}
