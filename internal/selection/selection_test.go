package selection

import (
	"reflect"
	"testing"
)

func checkCount(t *testing.T, s *Set) {
	t.Helper()
	if s.Count() != len(s.Indices()) {
		t.Fatalf("Count() = %d, len(Indices()) = %d", s.Count(), len(s.Indices()))
	}
	want := 0
	for i := 0; i < s.Len(); i++ {
		if s.IsSelected(i) {
			want++
		}
	}
	if want != s.Count() {
		t.Fatalf("IsSelected scan = %d, Count() = %d", want, s.Count())
	}
}

func TestToggle(t *testing.T) {
	s := New(5)
	if s.Count() != 0 || s.Len() != 5 {
		t.Fatalf("fresh set: count=%d len=%d", s.Count(), s.Len())
	}
	s.Toggle(2)
	if !s.IsSelected(2) || s.Count() != 1 {
		t.Fatalf("after Toggle(2): %v", s.Indices())
	}
	s.Toggle(2)
	if s.IsSelected(2) || s.Count() != 0 {
		t.Fatalf("after second Toggle(2): %v", s.Indices())
	}
	s.Toggle(99) // out of range, ignored
	s.Toggle(-1) // ignored
	checkCount(t, s)
}

func TestSelectRangeForwardBackward(t *testing.T) {
	s := New(10)
	s.Toggle(2) // anchor = 2
	s.SelectRange(5)
	if got := s.Indices(); !reflect.DeepEqual(got, []int{2, 3, 4, 5}) {
		t.Fatalf("forward range: %v", got)
	}
	checkCount(t, s)

	s.Toggle(9) // anchor = 9
	s.SelectRange(7)
	if got := s.Indices(); !reflect.DeepEqual(got, []int{2, 3, 4, 5, 7, 8, 9}) {
		t.Fatalf("backward range: %v", got)
	}
	checkCount(t, s)

	// Range adds only: re-running the same range must not unselect.
	s.SelectRange(7)
	if got := s.Indices(); !reflect.DeepEqual(got, []int{2, 3, 4, 5, 7, 8, 9}) {
		t.Fatalf("re-range changed selection: %v", got)
	}
	checkCount(t, s)
}

func TestSelectRangeKeepsAnchor(t *testing.T) {
	s := New(10)
	s.Toggle(1) // anchor = 1
	s.SelectRange(4)
	s.SelectRange(0) // anchor must still be 1
	if got := s.Indices(); !reflect.DeepEqual(got, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("anchor should stay at 1: %v", got)
	}
	checkCount(t, s)
}

func TestSelectRangeWithoutAnchor(t *testing.T) {
	s := New(10)
	s.SelectRange(3) // behaves like Toggle(3)
	if !s.IsSelected(3) || s.Count() != 1 {
		t.Fatalf("range without anchor: %v", s.Indices())
	}
	checkCount(t, s)
}

func TestSelectAllNoneInvert(t *testing.T) {
	s := New(6)
	s.SelectAll()
	if s.Count() != 6 {
		t.Fatalf("SelectAll count = %d", s.Count())
	}
	s.SelectAll() // idempotent
	if s.Count() != 6 {
		t.Fatalf("double SelectAll count = %d", s.Count())
	}
	checkCount(t, s)

	s.SelectNone()
	if s.Count() != 0 || len(s.Indices()) != 0 {
		t.Fatalf("SelectNone: %v", s.Indices())
	}
	checkCount(t, s)

	s.Toggle(0)
	s.Toggle(2)
	s.Invert()
	if got := s.Indices(); !reflect.DeepEqual(got, []int{1, 3, 4, 5}) {
		t.Fatalf("Invert: %v", got)
	}
	checkCount(t, s)
	s.Invert()
	if got := s.Indices(); !reflect.DeepEqual(got, []int{0, 2}) {
		t.Fatalf("double Invert: %v", got)
	}
	checkCount(t, s)
}

func TestSelectNoneClearsAnchor(t *testing.T) {
	s := New(10)
	s.Toggle(4)
	s.SelectNone()
	s.SelectRange(7) // no anchor → must behave like Toggle(7)
	if got := s.Indices(); !reflect.DeepEqual(got, []int{7}) {
		t.Fatalf("anchor not cleared: %v", got)
	}
	checkCount(t, s)
}

func TestIndicesAscending(t *testing.T) {
	s := New(20)
	for _, i := range []int{15, 3, 19, 3, 7, 0} {
		s.Toggle(i)
	}
	got := s.Indices()
	want := []int{0, 7, 15, 19} // 3 toggled twice = unselected
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Indices = %v, want %v", got, want)
	}
	checkCount(t, s)
}

func TestSetAnchorThenRange(t *testing.T) {
	s := New(5)
	s.SetAnchor(1)
	s.SelectRange(3)
	if got := s.Indices(); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("expected 1..3, got %v", got)
	}
	s.SetAnchor(100) // out of range ignored, anchor stays 1
	s.SelectRange(0) // additive: 0 added to existing 1..3
	if got := s.Indices(); len(got) != 4 || got[0] != 0 || got[3] != 3 {
		t.Fatalf("expected 0..3, got %v", got)
	}
}
