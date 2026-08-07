package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/coolvegan/go-hackernews/internal"
)

var (
	WORKERCOUNT         = 10
	OLDARTICLELOADCOUNT = 5000
	MINARTICLE          = 200
	// Alle X Minuten die Datenstruktur im Speicher verkleinern
	HALVEMEMORYDURATION = 1800
	SERVER              = fmt.Sprintf("localhost:7777")
	MCPSERVER           = fmt.Sprintf("localhost:13333")
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
	halftime := os.Getenv("HALFTIME")
	ht, err := strconv.Atoi(halftime)
	if err == nil {
		HALVEMEMORYDURATION = ht
	}
	preloaditems := os.Getenv("PRELOADITEMS")
	pl, err := strconv.Atoi(preloaditems)
	if err == nil {
		OLDARTICLELOADCOUNT = pl
	}

	var mu sync.RWMutex
	var watermarkmu sync.RWMutex
	var hackerNewsItems []*internal.Item
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
		internal.RunHackernewsMcp(MCPSERVER, &hackerNewsItems, &mu)
	}()

	go func() {
		//Alte Artikel laden
		waterMarkTicker := time.NewTicker(time.Second * 5)
		itemShowTimer := time.NewTicker(time.Second * 15)
		inHalfCutTimer := time.NewTicker(time.Minute * time.Duration(HALVEMEMORYDURATION))
		for {
			select {
			case <-ctx.Done():
				log.Println("Ending Application")
				return
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
				articleCount = len(hackerNewsItems)
				mu.RUnlock()
				log.Printf("%d Article from Hackernews in Memory.\n", articleCount)
			case <-inHalfCutTimer.C:
				mu.Lock()
				if len(hackerNewsItems) >= MINARTICLE {
					half := len(hackerNewsItems) / 2
					//Slice halbieren und Speicher freigeben
					hackerNewsItems = slices.Clone(hackerNewsItems[half:])
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
			hackerNewsItems = append(hackerNewsItems, r)
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
		json.NewEncoder(w).Encode(hackerNewsItems)
		mu.RUnlock()
	})

	//Watermark, fürs Polling, keine Lust auf SSE
	http.HandleFunc("/api/watermark", func(w http.ResponseWriter, r *http.Request) {
		watermarkmu.RLock()
		data, err := json.Marshal(watermark)
		watermarkmu.Unlock()
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, string(data))
	})
	log.Printf("Starting DEBUG-Server on Port %v", SERVER)
	log.Fatalln(http.ListenAndServe(SERVER, nil))
}
