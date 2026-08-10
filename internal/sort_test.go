package internal

import (
	"testing"
	"time"
)

func TestSortArticles(t *testing.T) {
	base := time.Now().Unix()
	mk := func(score int, ageMin int) *Item {
		return &Item{
			Score: score,
			Time:  base - int64(ageMin)*60,
		}
	}

	items := []*Item{
		mk(50, 120), // old, high score
		mk(5, 5),    // fresh, low score
		mk(20, 60),  // mid
		mk(100, 300), // oldest, highest score
	}

	// Default: score descending
	sortArticles(items, "score")
	want := []int{100, 50, 20, 5}
	for i, w := range want {
		if items[i].Score != w {
			t.Fatalf("score sort: items[%d].Score = %d, want %d", i, items[i].Score, w)
		}
	}

	// Newest: time descending (most recent first)
	sortArticles(items, "newest")
	wantNew := []int{5, 20, 50, 100}
	for i, w := range wantNew {
		if items[i].Score != w {
			t.Fatalf("newest sort: items[%d].Score = %d, want %d", i, items[i].Score, w)
		}
	}
}
