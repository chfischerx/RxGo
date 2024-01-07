package rxgo

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

// TestDefaultOptions verifies that multiple observers receive the same number of items
func TestDefaultOptions(t *testing.T) {

	subject := NewSubject()
	_, obs1 := subject.Subscribe()
	_, obs2 := subject.Subscribe()

	var wg sync.WaitGroup
	wg.Add(1)
	items := 10
	go func() {
		for i := 0; i < items; i++ {
			subject.Next(i)
		}
		wg.Done()
	}()

	itemCount1 := 0
	obs1.DoOnNext(func(i interface{}) {
		itemCount1++
	})

	itemCount2 := 0
	obs2.DoOnNext(func(i interface{}) {
		itemCount2++
	})

	wg.Wait()
	// short delay to give go routines time to process last element
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, items, itemCount1)
	assert.Equal(t, items, itemCount2)
}

// TestBackPressure verifies messages are dropped with a blocked observer
func TestBackPressure(t *testing.T) {

	subject := NewSubject(WithBackPressureStrategy(Drop))
	_, obs1 := subject.Subscribe()
	_, obs2 := subject.Subscribe()

	// first observer starts receiving
	itemCount1 := 0
	obs1.DoOnNext(func(i interface{}) {
		itemCount1++
	})

	var wg sync.WaitGroup
	items := 5

	// send first batch
	wg.Add(1)
	go func() {
		for i := 0; i < items; i++ {
			// slow down to avoid dropped messages
			time.Sleep(1 * time.Millisecond)
			subject.Next(i)
		}
		wg.Done()
	}()

	// wait for first batch of messages sent
	wg.Wait()

	// short delay to give go routines time to process last element
	time.Sleep(10 * time.Millisecond)
	// first observer should have received all items
	assert.Equal(t, items, itemCount1)

	// second observer starts receiving
	itemCount2 := 0
	obs2.DoOnNext(func(i interface{}) {
		itemCount2++
	})

	// send second batch
	wg.Add(1)
	go func() {
		for i := 0; i < items; i++ {
			// slow down to avoid dropped messages
			time.Sleep(1 * time.Millisecond)
			subject.Next(i + items)
		}
		wg.Done()
	}()
	// wait for second batch of messages sent
	wg.Wait()

	// short delay to give go routines time to process last element
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 2*items, itemCount1)
	assert.Equal(t, items, itemCount2)
}

// TestSubscriberBuffer verify no messages dropped with buffer attached
func TestSubscriberBuffer(t *testing.T) {

	subject := NewSubject(WithBufferedChannel(10), WithBackPressureStrategy(Drop))
	_, obs := subject.Subscribe()

	items := 10
	itemCount := 0
	obs.DoOnNext(func(i interface{}) {
		itemCount++
	})

	for i := 0; i < items; i++ {
		// short delay to let subscriber catch up
		time.Sleep(1 * time.Millisecond)
		subject.Next(i)
	}

	// short delay to give go routines time to process last element
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, items, itemCount)
}

func TestUnsubscribe(t *testing.T) {
	subject := NewSubject()
	sub, obs := subject.Subscribe()

	items := 10
	itemCount := 0
	obs.DoOnNext(func(i interface{}) {
		itemCount += 1
	})

	for i := 0; i < items; i++ {
		subject.Next(i)
	}

	// items after unsubscribe will be lost
	sub.Unsubscribe()
	for i := 0; i < 5; i++ {
		subject.Next(i)
	}

	assert.Equal(t, items, itemCount)
}

func TestReceiveError(t *testing.T) {

	subject := NewSubject()
	_, obs := subject.Subscribe()

	err := errors.New("test")

	obs.DoOnNext(func(i interface{}) {
		fmt.Printf("next %v", i)
	})

	var errRcvd error
	obs.DoOnError(func(e error) {
		fmt.Printf("error received: %v", e)
		errRcvd = e
	})

	subject.Error(err)

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, err, errRcvd)
}

func TestCompletion(t *testing.T) {

	subject := NewSubject()
	_, obs := subject.Subscribe()

	obs.DoOnNext(func(i interface{}) {
		fmt.Printf("next %v", i)
	})

	obs.DoOnCompleted(func() {
		fmt.Println("completed")
	})

	subject.Complete()
}

// TestConnectableSubject subscriber only starts receiving after connecting
func xTestConnectableSubject(t *testing.T) {
	subject := NewSubject(WithPublishStrategy())
	_, obs := subject.Subscribe()

	items := 5
	itemCount := 0
	var lastItem interface{}
	var wg sync.WaitGroup

	obs.DoOnNext(func(i interface{}) {
		fmt.Printf("received item %v\n", i)
		itemCount++
		lastItem = i
	})

	// before connect will be lost
	wg.Add(1)
	go func() {
		for i := 0; i < 5; i++ {
			subject.Next(i)
		}
		wg.Done()
	}()
	wg.Wait()

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, itemCount)
	// after connect counter shall go up
	obs.Connect(context.Background())

	wg.Add(1)
	go func() {
		for i := 0; i < items; i++ {
			subject.Next(i + items)
		}
		wg.Done()
	}()
	wg.Wait()

	// short delay to give go routines time to process last element
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, items, itemCount)
	assert.Equal(t, lastItem, 10)
}
