package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

func (f *Fetcher) fetchComments(myItem *Item, itemsUri string, wg *sync.WaitGroup) {
	defer wg.Done()
	for _, kid := range myItem.Kids {
		childMap, err := fetch(kid, itemsUri)
		if err != nil {
			break
		}
		if childMap.Kids != nil {
			wg.Add(1)
			go f.fetchComments(childMap, itemsUri, wg)
		}
		myItem.Comments = append(myItem.Comments, childMap)
	}
}

// GET ARTICLE DATA BY COUNT DESCENDING FROM NEWEST TO LATEST ARTICLE
func (f *Fetcher) FetchItemsConcurrent(articleCount int) ([]*Item, error) {
	data, err := f.fetchData(f.topstoryUri)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if articleCount > len(data) {
		articleCount = len(data)
	}
	numJobs := articleCount
	jobs := make(chan int, numJobs)
	results := make(chan WorkResult, articleCount)
	workerCount := f.workerCount
	if articleCount < workerCount {
		workerCount = articleCount
	}
	for w := 1; w <= workerCount; w++ {
		go f.worker(w, jobs, results)
	}
	for i := 0; i < articleCount; i++ {
		jobs <- data[i]
	}
	close(jobs)
	var resultsList []*Item
	for a := 1; a <= numJobs; a++ {
		r := <-results
		if r.Err != nil {
			log.Println(r.Err)
		}
		resultsList = append(resultsList, &r.Item)
	}
	return resultsList, nil
}

func (f *Fetcher) FetchFoo(articleInputChan <-chan int) <-chan *Item {
	res := make(chan *Item, f.workerCount*4)
	go func() {
		for aid := range articleInputChan {
			f.jobs <- aid
		}
	}()
	go func() {
		for wr := range f.results {
			if wr.Err != nil {
				log.Println(wr.Err)
			}
			res <- &wr.Item
		}
	}()
	return res
}

// Fetch Items from Hackernews concurrently. If no articleIDs is given. The result-item will fetch the latest avaiable article.
func (f *Fetcher) FetchItemsConcurrentlyByIDs(articleIDs ...int) ([]*Item, error) {
	latestId, err := f.NewestArticleID()
	if len(articleIDs) == 0 {
		articleIDs = append(articleIDs, latestId)
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}
	for articleID := range articleIDs {
		if articleID > latestId {
			errMsg := fmt.Sprintf("ArticleID %d is invalid!", articleID)
			log.Printf(errMsg, articleID)
			return nil, errors.New(errMsg)
		}
	}
	numJobs := len(articleIDs)
	jobs := make(chan int, numJobs)
	results := make(chan WorkResult, numJobs)

	workerCount := f.workerCount
	if f.workerCount > len(articleIDs) {
		workerCount = len(articleIDs)
	}

	for w := 1; w <= workerCount; w++ {
		go f.worker(w, jobs, results)
	}
	for i := 0; i < len(articleIDs); i++ {
		jobs <- articleIDs[i]
	}
	close(jobs)
	var resultsList []*Item
	for a := 1; a <= numJobs; a++ {
		r := <-results
		if r.Err != nil {
			log.Println(r.Err)
		}
		resultsList = append(resultsList, &r.Item)
	}
	return resultsList, nil
}

// GET TOPSTORIES
func (f *Fetcher) fetchData(url string) ([]int, error) {
	res, err := http.Get(f.topstoryUri)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer res.Body.Close()
	var result []int
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, nil
}

func (f *Fetcher) NewestArticleID() (int, error) {
	res, err := http.Get(f.newestIdUri)
	if err != nil {
		log.Println(err)
		return -1, err
	}
	data, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		log.Println(err)
		return -1, err
	}
	var result int
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Println(err)
		return -1, err
	}
	return result, nil
}

func (f *Fetcher) GetLatestArticle() (Item, error) {
	data, err := f.FetchItemsConcurrentlyByIDs()
	if err != nil {
		log.Println(err)
		return Item{}, err
	}
	return *data[0], nil
}
