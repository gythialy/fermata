package breakwin

import (
	"testing"
)

func TestEntranceCurves(t *testing.T) {
	if got := fadeAt(1, 0); got != 0 {
		t.Errorf("fadeAt(1, 0) = %v, want 0", got)
	}
	if got := fadeAt(1, entranceFade); got != 1 {
		t.Errorf("fadeAt(1, end) = %v, want 1", got)
	}
	if got := fadeAt(1, 2*entranceFade); got != 1 {
		t.Errorf("fadeAt(1, past) = %v, want 1", got)
	}
	if got := fadeAt(0.8, entranceFade); got != 0.8 {
		t.Errorf("fadeAt(0.8, end) = %v, want 0.8", got)
	}
	if fadeAt(1, entranceFade/4) >= fadeAt(1, entranceFade/2) {
		t.Error("the panel fade is not monotonic")
	}

	if got := riseAt(0); got != 6 {
		t.Errorf("riseAt(0) = %v, want 6", got)
	}
	if got := riseAt(contentRise); got != 0 {
		t.Errorf("riseAt(end) = %v, want 0", got)
	}
	if got := riseAt(contentRise / 2); got <= 0 || got >= 6 {
		t.Errorf("riseAt(mid) = %v, want between 0 and 6", got)
	}
	if riseAt(contentRise/4) <= riseAt(contentRise/2) {
		t.Error("the content rise does not settle")
	}
}
