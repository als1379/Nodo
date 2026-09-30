package flashcard

import "testing"

func TestTargetAfterSession(t *testing.T) {
	for _, tc := range []struct {
		target         float64
		answers, known int
		want           float64
	}{
		{15, 9, 9, 15}, {15, 20, 18, 17}, {15, 20, 12, 15}, {15, 20, 8, 13}, {99, 20, 20, 100}, {1, 20, 0, 0},
	} {
		if got := TargetAfterSession(tc.target, tc.answers, tc.known); got != tc.want {
			t.Errorf("%+v: got %v", tc, got)
		}
	}
}
