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

// InitWorker takes JobId as INT. A fetched Job could be an master or part of master node.
// If a master node is found - it will lock there JobID to the worklog via f.Lock(jobId)
// If the node is a child node. We will lock the ParentId (MasterNodeId) in the worklog
// To stop two worker processing on the same MasterNode and all his child-nodes
func (f *Fetcher) InitWorker(jobs <-chan int, results chan<- WorkResult) {
	for workerId := 1; workerId <= f.workerCount; workerId++ {
		go func(workerId int, jobs <-chan int, results chan<- WorkResult) {
			var wg sync.WaitGroup
			log.Printf("Worker started %d", workerId)
			for jobId := range jobs {
				// log.Printf("Worker %d bearbeitet Job %d", workerId, jobId)
				dataMap, err := fetch(jobId, f.itemsUri)
				if err != nil {
					log.Println(err)
					continue
				}
				//only take new articles
				if dataMap.Type == "story" {
					exclusiveJobId := f.Lock(jobId)
					defer f.Unlock(jobId)
					if !exclusiveJobId {
						log.Printf("Worker %d could not get jobId lock for job %d\n", workerId, jobId)
						continue
					}
					if dataMap.Dead || dataMap.Id == 0 || (dataMap.Url == "" && dataMap.Title == "") {
						log.Printf("Worker %d ignores job %d\n", workerId, jobId)
						// log.Printf("Worker %d ignoriert Job %d - Child[%v] Dead[%v] ZeroId[%v]", workerId, jobId, dataMap.Parent != 0, dataMap.Dead, dataMap.Id == 0)
						continue
					}
				}
				if dataMap.Type == "comment" {
					parentId := dataMap.Parent
					exclusiveJobId := f.Lock(parentId)
					defer f.Unlock(parentId)
					if !exclusiveJobId {
						log.Printf("Worker %d could not get parentId lock for parentId %d of job %d\n", workerId, parentId, jobId)
						continue
					}
					for i := 0; i < 10; i++ {
						log.Printf("Worker %d tries to find the story of comment of parentId %d \n", workerId, parentId)
						dataMap, err = fetch(parentId, f.itemsUri)
						if err != nil {
							log.Println(err)
							continue
						}
						if dataMap.Type == "story" {
							log.Printf("Worker %d found the story of comment of jobId %d \n", workerId, jobId)
							break
						}
						parentId = dataMap.Parent
						exclusiveJobId = f.Lock(parentId)
						defer f.Unlock(parentId)
						if !exclusiveJobId {
							log.Printf("Worker %d could not get parentId lock for parentId %d of job %d\n", workerId, parentId, jobId)
							continue
						}
					}
					if dataMap.Parent != 0 {
						log.Printf("Worker %d ignores parentId %d\n", workerId, parentId)
						continue
					}
					log.Printf("Worker %d fetching parentId %d of job %d\n", workerId, parentId, jobId)
				}
				if f.downloadComments {
					wg.Add(1)
					f.fetchComments(dataMap, f.itemsUri, &wg)
				}
				wg.Wait()
				log.Printf("Worker %d worked successfully on ItemID %d.\n", workerId, jobId)
				if dataMap.Dead || dataMap.Id == 0 || (dataMap.Url == "" && dataMap.Title == "") {
					log.Printf("Worker %d ignores job %d\n", workerId, jobId)
					// log.Printf("Worker %d ignoriert Job %d - Child[%v] Dead[%v] ZeroId[%v]", workerId, jobId, dataMap.Parent != 0, dataMap.Dead, dataMap.Id == 0)
					continue
				}
				results <- WorkResult{Item: *dataMap, Err: err}

			}
		}(workerId, jobs, results)
	}
}
