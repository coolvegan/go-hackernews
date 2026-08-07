package internal

import (
	"log"
	"sync"
)

func (f *Fetcher) worker(id int, jobs <-chan int, results chan<- WorkResult) {
	var wg sync.WaitGroup
	for jobId := range jobs {
		dataMap, err := fetch(jobId, f.itemsUri)
		if err != nil {
			log.Println(err)
			continue
		}
		if f.downloadComments {
			go f.fetchComments(dataMap, f.itemsUri, &wg)
		}
		wg.Wait()
		log.Printf("Worker %d arbeitete erfolgreich für ItemID %d.\n", id, jobId)
		results <- WorkResult{Item: *dataMap, Err: err}
	}
}

func (f *Fetcher) InitWorker(jobs <-chan int, results chan<- WorkResult) {
	for i := 1; i <= f.workerCount; i++ {
		go func(workerId int, jobs <-chan int, results chan<- WorkResult) {
			var wg sync.WaitGroup
			log.Printf("Worker gestarted %d", workerId)
			for jobId := range jobs {
				// log.Printf("Worker %d bearbeitet Job %d", workerId, jobId)

				dataMap, err := fetch(jobId, f.itemsUri)
				if err != nil {
					log.Println(err)
					continue
				}
				//only take new threads
				if dataMap.Parent != 0 || dataMap.Dead || dataMap.Id == 0 {
					// log.Printf("Worker %d ignoriert Job %d - Child[%v] Dead[%v] ZeroId[%v]", workerId, jobId, dataMap.Parent != 0, dataMap.Dead, dataMap.Id == 0)
					continue
				}
				if f.downloadComments {
					wg.Add(1)
					f.fetchComments(dataMap, f.itemsUri, &wg)
				}
				wg.Wait()
				log.Printf("Worker %d arbeitete erfolgreich für ItemID %d.\n", workerId, jobId)
				results <- WorkResult{Item: *dataMap, Err: err}

			}
		}(i, jobs, results)
	}
}
