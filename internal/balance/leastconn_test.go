package balance

import "testing"

func TestLeastConnectionsSelectsLowestActiveCount(t *testing.T) {
	backends := testBackends("a", "b", "c")
	releaseA := backends[0].Acquire()
	releaseC1 := backends[2].Acquire()
	releaseC2 := backends[2].Acquire()
	defer releaseA()
	defer releaseC1()
	defer releaseC2()
	got, err := new(LeastConnections).Next(backends)
	if err != nil || got.Name() != "b" {
		t.Fatalf("selection = %v, %v", got, err)
	}
}

func TestLeastConnectionsRotatesTies(t *testing.T) {
	backends := testBackends("a", "b")
	lc := new(LeastConnections)
	for _, want := range []string{"a", "b", "a"} {
		got, err := lc.Next(backends)
		if err != nil || got.Name() != want {
			t.Fatalf("selection = %v, %v; want %s", got, err, want)
		}
	}
}
