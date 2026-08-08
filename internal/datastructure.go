package internal

import (
	"sync"
	"time"
)

type ArticleSummary struct {
	Title string `json:"title"`
	Text  string `json:"text"`
	Url   string `json:"url"`
	Score int    `json:"score"`
}

// ArticleView is the full item without the (potentially huge) embedded
// comment tree. CommentCount replaces the recursive Comments field so a
// "full" result stays small no matter how many comments a story has.
type ArticleView struct {
	Id           int                     `json:"id"`
	Score        int                     `json:"score"`
	Time         int64                   `json:"time"`
	Title        string                  `json:"title"`
	Text         string                  `json:"text"`
	Type         string                  `json:"type"`
	By           string                  `json:"by"`
	Url          string                  `json:"url"`
	Parent       int                     `json:"parent"`
	Dead         bool                    `json:"dead"`
	CommentCount int `json:"comment_count"`
	// Comments maps a parent id to its direct child comment ids (no text).
	// The story id is the key for top-level comments, each comment id is the
	// key for its replies. Text is fetched separately via the comments tool.
	Comments map[int][]int `json:"comments"`
}

// CommentView is one comment with its text. Returned only for explicitly
// requested ids, so the response stays small unless the client asks for
// content.
type CommentView struct {
	Id     int    `json:"id"`
	Parent int    `json:"parent"`
	By     string `json:"by"`
	Text   string `json:"text"`
}

// CommentMap builds parent-id -> []direct child ids. The story id is the key
// for top-level comments; every comment id is the key for its own replies.
// It carries only ids (no text), so a story's whole thread structure stays
// tiny. Returns (idMap, totalCommentCount).
func (i *Item) CommentMap() (map[int][]int, int) {
	m := make(map[int][]int)
	count := 0
	var walk func(parent int, items []*Item)
	walk = func(parent int, items []*Item) {
		for _, c := range items {
			m[parent] = append(m[parent], c.Id)
			count++
			if len(c.Comments) > 0 {
				walk(c.Id, c.Comments)
			}
		}
	}
	walk(i.Id, i.Comments)
	return m, count
}

// CommentLookup flattens the whole comment tree into an id -> CommentView
// map so the handler can return the text for any explicitly requested ids.
func (i *Item) CommentLookup() map[int]CommentView {
	out := make(map[int]CommentView)
	var walk func(items []*Item)
	walk = func(items []*Item) {
		for _, c := range items {
			out[c.Id] = CommentView{Id: c.Id, Parent: c.Parent, By: c.By, Text: c.Text}
			if len(c.Comments) > 0 {
				walk(c.Comments)
			}
		}
	}
	walk(i.Comments)
	return out
}

type Item struct {
	Id        int    `json:"id"`
	Score     int    `json:"score"`
	Time      int64  `json:"time"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Type      string `json:"type"`
	By        string `json:"by"`
	Kids      []int  `json:"kids"`
	Url       string `json:"url"`
	Parent    int    `json:"parent"`
	Dead      bool   `json:"dead"`
	Comments  []*Item
	FetchedAt time.Time
	// Descendants any `json:"descendants"`
}

func (i *Item) Created() time.Time {
	return time.Unix(i.Time, 0)
}

type Fetcher struct {
	workerCount      int
	itemsUri         string
	topstoryUri      string
	newestIdUri      string
	downloadComments bool
	jobs             chan int
	results          chan WorkResult
	worklog          map[int]struct{}
	wrklmu           sync.RWMutex
}

func NewFetcher(workerCount int) *Fetcher {
	jobs := make(chan int, workerCount)
	results := make(chan WorkResult, workerCount)
	worklog := make(map[int]struct{})
	f := Fetcher{itemsUri: ITEMSURL, topstoryUri: TOPSTORIES, newestIdUri: MAXITEMURL, workerCount: workerCount, downloadComments: true, jobs: jobs, results: results, worklog: worklog}
	f.InitWorker(jobs, results)
	return &f
}

type WorkResult struct {
	Item Item
	Err  error
}

func (f *Fetcher) CommentDownload(option bool) {
	f.downloadComments = option
}

func (f *Fetcher) Lock(jobId int) bool {
	f.wrklmu.Lock()
	defer f.wrklmu.Unlock()
	_, jobIdIsAlreadyInProcess := f.worklog[jobId]
	if jobIdIsAlreadyInProcess {
		return false
	}
	f.worklog[jobId] = struct{}{}
	return true
}

func (f *Fetcher) Unlock(jobId int) {
	f.wrklmu.Lock()
	delete(f.worklog, jobId)
	f.wrklmu.Unlock()
}
