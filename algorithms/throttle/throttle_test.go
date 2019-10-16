package throttle_test

import (
	"sync"
	"testing"
	"time"

	"github.com/go-phorce/dolly/algorithms/throttle"
	"github.com/stretchr/testify/assert"
)

func Test_NoPings(t *testing.T) {
	var wg sync.WaitGroup
	throttle := throttle.NewThrottle(time.Millisecond, false)

	count := 0

	wg.Add(1)
	go func() {
		defer wg.Done()
		for throttle.Next() {
			count += 1
		}
	}()

	time.Sleep(10 * time.Millisecond)
	throttle.Stop()

	wg.Wait()

	assert.Equal(t, 0, count)
}

func Test_MultiPingInOnePeriod(t *testing.T) {
	var wg sync.WaitGroup

	throttle := throttle.NewThrottle(time.Millisecond, false)
	count := 0

	wg.Add(1)
	go func() {
		defer wg.Done()
		for throttle.Next() {
			count += 1
		}
	}()

	for i := 0; i < 5; i++ {
		throttle.Trigger()
	}

	time.Sleep(5 * time.Millisecond)

	throttle.Stop()

	wg.Wait()

	assert.Equal(t, 1, count)
}

func Test_MultiPingInMultiplePeriod(t *testing.T) {
	var wg sync.WaitGroup

	throttle := throttle.NewThrottle(4*time.Millisecond, false)
	count := 0

	wg.Add(1)
	go func() {
		defer wg.Done()
		for throttle.Next() {
			count += 1
		}
	}()

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		throttle.Trigger()
	}

	time.Sleep(5 * time.Millisecond)

	throttle.Stop()

	wg.Wait()

	assert.True(t, count >= 2 && count <= 4)
}

func Test_TrailingMultiPingInOnePeriod(t *testing.T) {
	var wg sync.WaitGroup

	throttle := throttle.NewThrottle(4*time.Millisecond, true)
	count := 0

	cond := sync.NewCond(&sync.Mutex{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for throttle.Next() {
			count += 1
			cond.Broadcast()
		}
	}()

	throttle.Trigger()

	cond.L.Lock()
	cond.Wait()
	throttle.Trigger()
	cond.L.Unlock()

	throttle.Trigger()
	time.Sleep(time.Millisecond)
	throttle.Trigger()
	time.Sleep(time.Millisecond)
	throttle.Trigger()

	time.Sleep(5 * time.Millisecond)

	throttle.Stop()

	wg.Wait()

	assert.True(t, count >= 2 && count <= 4)
}

func Test_ThrottleFunc(t *testing.T) {
	count := 0

	throttle := throttle.Func(time.Millisecond, false, func() {
		count += 1
	})

	for i := 0; i < 5; i++ {
		throttle.Trigger()
	}

	time.Sleep(5 * time.Millisecond)

	throttle.Stop()

	assert.Equal(t, 1, count)
}
