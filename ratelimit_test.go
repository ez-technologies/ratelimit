package ratelimit

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBurstThenRefusedThenRefilled(t *testing.T) {
	l := New(time.Hour)
	clock := time.Unix(1000, 0)
	l.now = func() time.Time { return clock }
	for i := 0; i < 3; i++ {
		assert.True(t, l.Allow("a", time.Second, 3), "burst %d", i)
	}
	assert.False(t, l.Allow("a", time.Second, 3))
	assert.True(t, l.Allow("b", time.Second, 3), "another key has its own bucket")
	clock = clock.Add(time.Second)
	assert.True(t, l.Allow("a", time.Second, 3), "one more after `every`")
	assert.False(t, l.Allow("a", time.Second, 3))
}

func TestFillingTheTableDoesNotResetALimit(t *testing.T) {
	l := New(time.Hour)
	for i := 0; i < 5; i++ {
		assert.True(t, l.Allow("global", time.Hour, 5))
	}
	assert.False(t, l.Allow("global", time.Hour, 5))
	for i := 0; i < 20000; i++ {
		l.Allow(fmt.Sprintf("fake-%d", i), time.Second, 5)
	}
	assert.Equal(t, 20001, l.Len(), "volume alone drops nothing")
	assert.False(t, l.Allow("global", time.Hour, 5), "the limit still holds")
}

func TestIdleEntriesExpireOnly(t *testing.T) {
	l := New(time.Minute)
	clock := time.Unix(1000, 0)
	l.now = func() time.Time { return clock }
	l.Allow("old", time.Second, 1)
	l.Allow("slow", time.Hour, 1) // takes an hour to refill: must outlive the idle time
	clock = clock.Add(2 * time.Minute)
	l.Allow("new", time.Second, 1)
	assert.Equal(t, 2, l.Len(), "old is gone, slow and new remain")
	assert.False(t, l.Allow("slow", time.Hour, 1), "slow is still limited")
}
