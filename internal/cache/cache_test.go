package cache

import (
	"net/http"
	"testing"
	"time"
)

func TestCacheReturnsDefensiveCopyAndExpires(t *testing.T) {
	c := New(2, 1024)
	now := time.Unix(100, 0)
	c.now = func() time.Time { return now }
	if !c.Set("one", Response{Status: 200, Header: http.Header{"X-Test": {"original"}}, Body: []byte("body")}, time.Second) {
		t.Fatal("set failed")
	}
	got, ok := c.Get("one")
	if !ok {
		t.Fatal("cache miss")
	}
	got.Header.Set("X-Test", "changed")
	got.Body[0] = 'X'
	again, _ := c.Get("one")
	if again.Header.Get("X-Test") != "original" || string(again.Body) != "body" {
		t.Fatal("cached value was mutated")
	}
	now = now.Add(time.Second)
	if _, ok := c.Get("one"); ok || c.Len() != 0 {
		t.Fatal("expired entry remained")
	}
}

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	c := New(2, 1024)
	c.Set("one", Response{Header: make(http.Header), Body: []byte("1")}, time.Minute)
	c.Set("two", Response{Header: make(http.Header), Body: []byte("2")}, time.Minute)
	c.Get("one")
	c.Set("three", Response{Header: make(http.Header), Body: []byte("3")}, time.Minute)
	if _, ok := c.Get("two"); ok {
		t.Fatal("least-recent item was not evicted")
	}
	if _, ok := c.Get("one"); !ok {
		t.Fatal("recent item was evicted")
	}
}

func TestCacheRejectsEntryLargerThanByteLimit(t *testing.T) {
	c := New(2, 4)
	if c.Set("key", Response{Header: make(http.Header), Body: []byte("large")}, time.Minute) {
		t.Fatal("oversized entry accepted")
	}
}
