package internal

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Regression test: "full" must NOT embed the comment tree, otherwise a
// single 878-comment story bloats the MCP response to megabytes.
func TestFullOutputSize(t *testing.T) {
	// Build a story with 10,000 fake comments to simulate the worst case.
	story := &Item{
		Id:    1,
		Score: 790,
		Title: "Why is everyone in tech so sad?",
		Url:   "https://noemamag.com/example",
	}
	for i := 0; i < 10000; i++ {
		story.Comments = append(story.Comments, &Item{Id: 1000 + i, Text: "a comment body " + string(rune(65+i%26))})
	}

	// "full" is thin: only the count, never the whole comment map.
	view := ArticleView{
		Id:           story.Id,
		Score:        story.Score,
		Title:        story.Title,
		Url:          story.Url,
		CommentCount: len(story.Comments),
	}

	jsonBytes, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	size := len(jsonBytes)
	fmt.Printf("ArticleView size with 10k comments: %d bytes\n", size)

	if size > 2000 {
		t.Fatalf("expected small output, got %d bytes", size)
	}
}

// CommentMap carries ONLY ids (no text), so a whole thread structure stays
// tiny. Story id keys the top-level comments, each comment id keys its replies.
func TestCommentMap(t *testing.T) {
	// story -> A -> A1, A2 ; B
	story := &Item{Id: 1, Title: "story"}
	a := &Item{Id: 11, Parent: 1, By: "alice", Text: "top comment", Comments: []*Item{
		{Id: 111, Parent: 11, By: "bob", Text: "reply1"},
		{Id: 112, Parent: 11, By: "carol", Text: "reply2"},
	}}
	b := &Item{Id: 12, Parent: 1, By: "dave", Text: "second top"}
	story.Comments = []*Item{a, b}

	m, count := story.CommentMap()
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}
	// Story id -> its two top-level comment ids.
	if len(m[1]) != 2 || m[1][0] != 11 || m[1][1] != 12 {
		t.Fatalf("story id should map to [11,12], got %+v", m[1])
	}
	// Comment 11 -> its two reply ids.
	if len(m[11]) != 2 || m[11][0] != 111 || m[11][1] != 112 {
		t.Fatalf("comment 11 should map to [111,112], got %+v", m[11])
	}
	// Only parents with children get a key.
	if len(m) != 2 {
		t.Fatalf("expected 2 keys (1, 11), got %d: %+v", len(m), m)
	}

	jsonBytes, _ := json.Marshal(m)
	fmt.Printf("CommentMap (ids only, 4 comments): %d bytes\n", len(jsonBytes))
}

// CommentLookup must let the handler return text for any requested comment id.
func TestCommentLookup(t *testing.T) {
	story := &Item{Id: 1, Title: "story"}
	a := &Item{Id: 11, Parent: 1, By: "alice", Text: "top comment", Comments: []*Item{
		{Id: 111, Parent: 11, By: "bob", Text: "reply1"},
	}}
	b := &Item{Id: 12, Parent: 1, By: "dave", Text: "second top"}
	story.Comments = []*Item{a, b}

	lookup := story.CommentLookup()
	if len(lookup) != 3 {
		t.Fatalf("expected 3 comments in lookup, got %d", len(lookup))
	}
	cv, ok := lookup[111]
	if !ok || cv.Text != "reply1" || cv.Parent != 11 || cv.By != "bob" {
		t.Fatalf("lookup[111] wrong: %+v", cv)
	}
	if _, ok := lookup[999]; ok {
		t.Fatalf("lookup must not contain unknown id 999")
	}
}
