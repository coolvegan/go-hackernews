package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/coolvegan/go-hackernews/internal"
)

var (
	WORKERCOUNT         = 10
	OLDARTICLELOADCOUNT = 1000
	MINARTICLE          = 10
	// Alle X Minuten die Datenstruktur im Speicher verkleinern
	HOURSOLDERSINCEFETCH = 12
	SERVER               = "localhost:7777"
	MCPSERVER            = "localhost:13333"
	PERSISTENCEINTERVAL  = 5
)

func main() {
	srv := os.Getenv("SERVER")
	if srv != "" {
		SERVER = srv
	}
	mcpserver := os.Getenv("MCPSERVER")
	if mcpserver != "" {
		MCPSERVER = mcpserver
	}

	workercount := os.Getenv("WORKERCOUNT")
	wc, err := strconv.Atoi(workercount)
	if err == nil {
		WORKERCOUNT = wc
	}
	persistenceinterval := os.Getenv("PERSISTENCEINTERVAL")
	pi, err := strconv.Atoi(persistenceinterval)
	if err == nil {
		PERSISTENCEINTERVAL = pi
	}
	halftime := os.Getenv("HALFTIME")
	ht, err := strconv.Atoi(halftime)
	if err == nil {
		HOURSOLDERSINCEFETCH = ht
	}
	preloaditems := os.Getenv("PRELOADITEMS")
	pl, err := strconv.Atoi(preloaditems)
	if err == nil {
		OLDARTICLELOADCOUNT = pl
	}

	var mu sync.RWMutex
	var watermarkmu sync.RWMutex
	persistence := internal.NewFsPeristence()
	hackerNewsItemsMap, err := persistence.Fetch()
	if err != nil {
		hackerNewsItemsMap = make(internal.HNData)
		log.Println("Fresh start")
	}

	articleInputChan := make(chan int, WORKERCOUNT)
	defer close(articleInputChan)
	log.Println("Starting Hackernews Fetcher")
	//Todo in den Fetcher bringen
	ctx := context.Background()
	f := internal.NewFetcher(WORKERCOUNT)
	//Starte MCP Server
	latestId, err := f.NewestArticleID()
	if err != nil {
		log.Println(err)
	}
	watermarkmu.Lock()
	watermark := latestId - OLDARTICLELOADCOUNT
	watermarkmu.Unlock()
	go func() {
		internal.RunHackernewsMcp(MCPSERVER, hackerNewsItemsMap, &mu)
	}()

	go func() {
		waterMarkTicker := time.NewTicker(time.Second * 5)
		itemShowTimer := time.NewTicker(time.Second * 15)
		deleteOldItemsTimer := time.NewTicker(time.Minute * 5)
		PersistenceTimer := time.NewTicker(time.Minute * time.Duration(PERSISTENCEINTERVAL))
		for {
			select {
			case <-ctx.Done():
				persistence.Store(hackerNewsItemsMap)
				log.Println("Ending Application")
				return
			case <-PersistenceTimer.C:
				mu.Lock()
				persistence.Store(hackerNewsItemsMap)
				mu.Unlock()

			case <-waterMarkTicker.C:
				latestId, err := f.NewestArticleID()
				if err != nil {
					continue
				}
				watermarkmu.Lock()
				for aid := watermark + 1; aid <= latestId; aid++ {
					articleInputChan <- aid
				}
				watermark = latestId
				watermarkmu.Unlock()
				// log.Printf("Old watermark: %d New watermark: %d", watermark, latestId)
			case <-itemShowTimer.C:
				var articleCount int
				mu.RLock()
				articleCount = len(hackerNewsItemsMap)
				mu.RUnlock()
				log.Printf("%d Article from Hackernews in Memory.\n", articleCount)
			case <-deleteOldItemsTimer.C:
				mu.Lock()
				if len(hackerNewsItemsMap) >= MINARTICLE {
					for k, v := range hackerNewsItemsMap {
						if time.Since(v.FetchedAt) >= time.Hour*time.Duration(HOURSOLDERSINCEFETCH) {
							log.Printf("Deleting Node with ID %d", v.Id)
							delete(hackerNewsItemsMap, k)
						}
					}
				}
				mu.Unlock()

			}
		}
	}()
	//Hauptdatenstruktur füllen. Diese wird auch vom Web und Mcp Server genutzt
	go func() {
		log.Println("Initial Fetch started.")
		res := f.FetchFoo(articleInputChan)
		for r := range res {
			mu.Lock()
			hackerNewsItemsMap[r.Id] = r
			mu.Unlock()
		}
	}()
	//Debug View
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	//Json Items
	http.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		mu.RLock()
		json.NewEncoder(w).Encode(hackerNewsItemsMap)
		mu.RUnlock()
	})

	//Json Item by hashmap-key
	http.HandleFunc("/api/item/{id}", func(w http.ResponseWriter, r *http.Request) {
		hashMapIdStr := r.PathValue("id")
		hashMapId, err := strconv.Atoi(hashMapIdStr)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			fmt.Fprintln(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		mu.RLock()
		json.NewEncoder(w).Encode(hackerNewsItemsMap[hashMapId])
		mu.RUnlock()
	})

	http.HandleFunc("/api/itemkeys", func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		mu.RLock()
		defer mu.RUnlock()
		if len(hackerNewsItemsMap) == 0 {
			fmt.Fprintln(w, "[]")
			return
		}
		var result []int
		for k, _ := range hackerNewsItemsMap {
			result = append(result, k)
		}
		if len(result) == 0 {
		}
		sort.SliceStable(result, func(i, j int) bool {
			return result[i] >= result[j]
		})
		json.NewEncoder(w).Encode(result)
	})
	//Watermark, fürs Polling, keine Lust auf SSE
	http.HandleFunc("/api/watermark", func(w http.ResponseWriter, r *http.Request) {
		watermarkmu.RLock()
		data, err := json.Marshal(watermark)
		watermarkmu.RUnlock()
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, string(data))
	})
	log.Printf("Starting DEBUG-Server on Port %v", SERVER)
	log.Printf("Switching to DebugWriter - Set Environment-Variable DEBUG=TRUE for more Output %v", SERVER)
	log.SetOutput(&internal.DebugWriter{})
	log.Fatalln(http.ListenAndServe(SERVER, nil))
}
