package store

import (
	"fmt"
	"sync"
)

func ExampleResourcePool() {
	const poolSize = 3
	const workerCount = 5
	const resource = "I am a pooled resource!"

	pool, err := NewCircularPool(poolSize, poolSize, func() interface{} { return resource })
	if err != nil {
		fmt.Println(err)
		return
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(workerCount)

	for repeat := 0; repeat < workerCount; repeat++ {
		go func() {
			defer waitGroup.Done()

			// check out a resource
			sharedResource := pool.Get().(string)

			// use it
			fmt.Println(sharedResource)

			// return it to the pool
			pool.Put(sharedResource)
		}()
	}

	waitGroup.Wait()

	// Output:
	// I am a pooled resource!
	// I am a pooled resource!
	// I am a pooled resource!
	// I am a pooled resource!
	// I am a pooled resource!
}
