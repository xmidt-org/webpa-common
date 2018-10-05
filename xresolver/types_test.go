package xresolver

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"net"
	"strconv"
	"testing"
)

func TestRoundRobinOperations(t *testing.T) {
	assert := assert.New(t)

	balancer := NewRoundRobinBalancer()

	expected := net.IPAddr{IP: net.ParseIP("127.0.0.1")}

	records, err := balancer.Get()
	assert.Error(err)
	assert.Empty(records)

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
}

func TestRoundRobinOrder(t *testing.T) {
	assert := assert.New(t)

	balancer := NewRoundRobinBalancer()

	localAddress := net.IPAddr{IP: net.ParseIP("127.0.0.1")}
	googleAddres := net.IPAddr{IP: net.ParseIP("8.8.8.8")}
	addressOnes := net.IPAddr{IP: net.ParseIP("1.1.1.1")}

	balancer.Add(localAddress)
	balancer.Add(googleAddres)
	balancer.Add(addressOnes)

	records, err := balancer.Get()
	assert.NoError(err)
	assert.Equal([]net.IPAddr{localAddress, googleAddres, addressOnes}, records, "records are assumed to be the order in which added")

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal([]net.IPAddr{googleAddres, addressOnes, localAddress}, records, "records should rotate on another Get()")

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal([]net.IPAddr{addressOnes, localAddress, googleAddres}, records, "records should rotate on another Get()")

	err = balancer.Remove(googleAddres)
	assert.NoError(err)

	records, err = balancer.Get()
	assert.NoError(err)
	assert.Equal([]net.IPAddr{localAddress, addressOnes}, records, "records should rotate on another Get()")
}

/**
Processor Speed: 2.8 GHz
Number of Processors: 1
Total Number of Cores: 4
L2 Cache (per Core): 256 KB
L3 Cache: 6 MB
Memory: 16 GB
Version: OS X 10.13.5

BenchmarkRoundRobinAdd/add/1-8         	10000000	       213 ns/op	      40 B/op	       3 allocs/op
BenchmarkRoundRobinAdd/add/2-8         	 3000000	       430 ns/op	      80 B/op	       6 allocs/op
BenchmarkRoundRobinAdd/add/4-8         	 2000000	       945 ns/op	     160 B/op	      12 allocs/op
BenchmarkRoundRobinAdd/add/8-8         	 1000000	      2224 ns/op	     320 B/op	      24 allocs/op
BenchmarkRoundRobinAdd/add/16-8        	  300000	      3879 ns/op	     640 B/op	      48 allocs/op
BenchmarkRoundRobinAdd/add/32-8        	  200000	      8129 ns/op	    1280 B/op	      96 allocs/op
BenchmarkRoundRobinAdd/add/64-8        	  100000	     16227 ns/op	    2560 B/op	     192 allocs/op
BenchmarkRoundRobinAdd/add/128-8       	   50000	     33005 ns/op	    5344 B/op	     412 allocs/op
BenchmarkRoundRobinAdd/add/256-8       	   20000	     67932 ns/op	   11490 B/op	     924 allocs/op
BenchmarkRoundRobinAdd/add/512-8       	   10000	    137234 ns/op	   22984 B/op	    1848 allocs/op
BenchmarkRoundRobinAdd/add/1024-8      	    5000	    285058 ns/op	   45985 B/op	    3696 allocs/op
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
					balancer.Add(net.IPAddr{IP: IPv4Address(int64(index))})
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

BenchmarkRoundRobinRemove/remove/1-8   	 2000000	       873 ns/op	      32 B/op	       3 allocs/op
BenchmarkRoundRobinRemove/remove/2-8   	 1000000	      1228 ns/op	      48 B/op	       6 allocs/op
BenchmarkRoundRobinRemove/remove/4-8   	 1000000	      1935 ns/op	      96 B/op	      12 allocs/op
BenchmarkRoundRobinRemove/remove/8-8   	  500000	      3525 ns/op	     192 B/op	      24 allocs/op
BenchmarkRoundRobinRemove/remove/16-8  	  200000	      8811 ns/op	     384 B/op	      48 allocs/op
BenchmarkRoundRobinRemove/remove/32-8  	  100000	     22149 ns/op	     768 B/op	      96 allocs/op
BenchmarkRoundRobinRemove/remove/64-8  	   20000	     64724 ns/op	    1536 B/op	     192 allocs/op
BenchmarkRoundRobinRemove/remove/128-8 	   10000	    207113 ns/op	    3744 B/op	     384 allocs/op
BenchmarkRoundRobinRemove/remove/256-8 	    2000	    782009 ns/op	    9888 B/op	     768 allocs/op
BenchmarkRoundRobinRemove/remove/512-8 	     500	   2855754 ns/op	   19776 B/op	    1536 allocs/op
BenchmarkRoundRobinRemove/remove/1024-8      100	  10627272 ns/op	   39552 B/op	    3072 allocs/op
*/

func BenchmarkRoundRobinRemove(b *testing.B) {
	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("remove/%d", n), func(b *testing.B) {
			b.ReportAllocs()

			balancer := NewRoundRobinBalancer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				records := make([]net.IP, n)
				for index := 0; index < n; index++ {
					ip := IPv4Address(int64(index))
					records[index] = ip
					balancer.Add(net.IPAddr{IP: ip})
				}
				b.StartTimer()
				for index := 0; index < n; index++ {
					err := balancer.Remove(net.IPAddr{IP: records[index]})
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

BenchmarkRoundRobinGet/get/1-8                  	10000000	       234 ns/op	      48 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/2-8                  	 5000000	       285 ns/op	      80 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/4-8                  	 5000000	       358 ns/op	     160 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/8-8                  	 3000000	       460 ns/op	     320 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/16-8                 	 2000000	       810 ns/op	     640 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/32-8                 	 1000000	      1497 ns/op	    1280 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/64-8                 	  500000	      2683 ns/op	    2688 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/128-8                	  300000	      5052 ns/op	    5376 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/256-8                	  200000	     10742 ns/op	   10240 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/512-8                	  100000	     19537 ns/op	   20480 B/op	       1 allocs/op
BenchmarkRoundRobinGet/get/1024-8               	   50000	     39192 ns/op	   40960 B/op	       1 allocs/op
*/

func BenchmarkRoundRobinGet(b *testing.B) {
	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("get/%d", n), func(b *testing.B) {
			b.ReportAllocs()

			balancer := NewRoundRobinBalancer()
			for index := 0; index < n; index++ {
				balancer.Add(net.IPAddr{IP: IPv4Address(int64(index))})
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

func IPv4Address(ipInt int64) net.IP {
	// need to do two bit shifting and “0xff” masking
	b0 := strconv.FormatInt((ipInt>>24)&0xff, 10)
	b1 := strconv.FormatInt((ipInt>>16)&0xff, 10)
	b2 := strconv.FormatInt((ipInt>>8)&0xff, 10)
	b3 := strconv.FormatInt((ipInt & 0xff), 10)

	return net.ParseIP(b0 + "." + b1 + "." + b2 + "." + b3)
}
