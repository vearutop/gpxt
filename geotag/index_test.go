package geotag

import (
	"math"
	"testing"
	"time"
)

func TestLookupInterpolate(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	idx := NewIndex(Options{
		MaxGap:      60 * time.Second,
		Interpolate: true,
	})
	idx.AddTrack([]Point{
		{Time: base.UnixNano(), Lat: 10, Lon: 20, Alt: 100},
		{Time: base.Add(10 * time.Second).UnixNano(), Lat: 20, Lon: 30, Alt: 200},
	})

	p, ok := idx.Lookup(base.Add(5 * time.Second))
	if !ok {
		t.Fatalf("expected ok")
	}
	if p.Lat != 15 || p.Lon != 25 {
		t.Fatalf("unexpected interpolation: %+v", p)
	}
}

func TestLookupMaxGap(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	idx := NewIndex(Options{
		MaxGap:      10 * time.Second,
		Interpolate: true,
	})
	idx.AddTrack([]Point{
		{Time: base.UnixNano(), Lat: 10, Lon: 20, Alt: 100},
	})

	_, ok := idx.Lookup(base.Add(30 * time.Second))
	if ok {
		t.Fatalf("expected not ok")
	}
}

func TestAltUnknown(t *testing.T) {
	if !math.IsNaN(float64(AltUnknown())) {
		t.Fatalf("expected NaN")
	}
}

func TestLookupWithClock(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	idx := NewIndex(Options{
		MaxGap:      60 * time.Second,
		Interpolate: false,
	})
	idx.AddTrack([]Point{
		{Time: base.UnixNano(), Lat: 10, Lon: 20, Alt: 100},
	})
	idx.AddClockSync("camA", base.Add(-10*time.Minute), base.Add(10*time.Minute), 30*time.Second)

	p, ok := idx.LookupWithClock(base.Add(-30*time.Second), "camA")
	if !ok {
		t.Fatalf("expected ok")
	}
	if p.Lat != 10 || p.Lon != 20 {
		t.Fatalf("unexpected lookup: %+v", p)
	}
}

func TestOffsetForLocation(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	idx := NewIndex(Options{})
	idx.AddTrack([]Point{
		{Time: base.UnixNano(), Lat: 10, Lon: 20},
		{Time: base.Add(10 * time.Second).UnixNano(), Lat: 11, Lon: 21},
	})

	cameraTime := base.Add(-30 * time.Second)
	offset, ok := idx.OffsetForLocation(cameraTime, 10.0, 20.0, 5)
	if !ok {
		t.Fatalf("expected ok")
	}
	if offset != 30*time.Second {
		t.Fatalf("unexpected offset: %v", offset)
	}
}
