package internal

import (
	"time"
)

type ArticleSummary struct {
	Title string `json:"title"`
	Text  string `json:"text"`
	Url   string `json:"url"`
	Score int    `json:"score"`
}

type Item struct {
	Id       int    `json:"id"`
	Score    int    `json:"score"`
	Time     int64  `json:"time"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	By       string `json:"by"`
	Kids     []int  `json:"kids"`
	Url      string `json:"url"`
	Parent   int    `json:"parent"`
	Dead     bool   `json:"dead"`
	Comments []*Item
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
}

func NewFetcher(workerCount int) *Fetcher {
	jobs := make(chan int, workerCount)
	results := make(chan WorkResult, workerCount)
	f := Fetcher{itemsUri: ITEMSURL, topstoryUri: TOPSTORIES, newestIdUri: MAXITEMURL, workerCount: workerCount, downloadComments: true, jobs: jobs, results: results}
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
