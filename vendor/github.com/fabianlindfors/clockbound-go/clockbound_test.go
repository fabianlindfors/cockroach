package clockbound

import (
	"fmt"
	"math"
	"testing"
)

func TestNow(t *testing.T) {
	clock, err := New()
	if err != nil {
		t.Error(err)
	}

	bounds, err := clock.Now()
	if err != nil {
		t.Error(err)
	}

	if bounds.Earliest > bounds.Latest {
		t.Error("expected earliest timestamp to be earlier than latest")
	}

	fmt.Printf("Earliest: %d, Latest: %d\n", bounds.Earliest, bounds.Latest)
}

func TestBefore(t *testing.T) {
	clock, err := New()
	if err != nil {
		t.Error(err)
	}

	before, err := clock.Before(0)
	if err != nil {
		t.Error(err)
	}
	if !before {
		t.Fatal("expected timestamp 0 to be before current time")
	}

	before, err = clock.Before(math.MaxUint64)
	if err != nil {
		t.Error(err)
	}
	if before {
		t.Fatal("expected maximum timestamp to not be before current time")
	}
}

func TestAfter(t *testing.T) {
	clock, err := New()
	if err != nil {
		t.Error(err)
	}

	after, err := clock.After(0)
	if err != nil {
		t.Error(err)
	}
	if after {
		t.Fatal("expected timestamp 0 to not be after current time")
	}

	after, err = clock.After(math.MaxUint64)
	if err != nil {
		t.Error(err)
	}
	if !after {
		t.Fatal("expected maximum timestamp to be after current time")
	}
}
