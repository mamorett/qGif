// Package selection tracks which frames are selected. It is pure Go and
// independent of any GUI toolkit.
package selection

// Set is the single source of truth for frame selection.
type Set struct {
	selected []bool
	count    int
	anchor   int // -1 = none
}

// New returns an empty selection over n frames.
func New(n int) *Set {
	if n < 0 {
		n = 0
	}
	return &Set{selected: make([]bool, n), anchor: -1}
}

// Len returns the number of frames.
func (s *Set) Len() int { return len(s.selected) }

// Count returns the number of selected frames. Maintained incrementally.
func (s *Set) Count() int { return s.count }

// IsSelected reports whether frame i is selected. Out-of-range is false.
func (s *Set) IsSelected(i int) bool {
	return i >= 0 && i < len(s.selected) && s.selected[i]
}

// Toggle flips frame i and makes it the range anchor.
func (s *Set) Toggle(i int) {
	if i < 0 || i >= len(s.selected) {
		return
	}
	s.selected[i] = !s.selected[i]
	if s.selected[i] {
		s.count++
	} else {
		s.count--
	}
	s.anchor = i
}

// SetAnchor makes frame i the start of the next Shift+click range. Out of
// range indices are ignored.
func (s *Set) SetAnchor(i int) {
	if i >= 0 && i < len(s.selected) {
		s.anchor = i
	}
}

// SelectRange selects every frame between the anchor and i inclusive, in
// either direction, adding to the current selection. The anchor is not
// changed. Without an anchor it behaves like Toggle(i).
func (s *Set) SelectRange(i int) {
	if i < 0 || i >= len(s.selected) {
		return
	}
	if s.anchor < 0 {
		s.Toggle(i)
		return
	}
	lo, hi := s.anchor, i
	if lo > hi {
		lo, hi = hi, lo
	}
	for j := lo; j <= hi; j++ {
		if !s.selected[j] {
			s.selected[j] = true
			s.count++
		}
	}
}

// SelectAll selects every frame.
func (s *Set) SelectAll() {
	for i := range s.selected {
		if !s.selected[i] {
			s.selected[i] = true
			s.count++
		}
	}
}

// SelectNone clears the selection and the anchor.
func (s *Set) SelectNone() {
	for i := range s.selected {
		s.selected[i] = false
	}
	s.count = 0
	s.anchor = -1
}

// Invert flips the selection of every frame.
func (s *Set) Invert() {
	for i := range s.selected {
		s.selected[i] = !s.selected[i]
		if s.selected[i] {
			s.count++
		} else {
			s.count--
		}
	}
}

// Indices returns the selected frame indices in ascending order.
func (s *Set) Indices() []int {
	out := make([]int, 0, s.count)
	for i, on := range s.selected {
		if on {
			out = append(out, i)
		}
	}
	return out
}
