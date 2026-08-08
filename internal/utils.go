package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// GET ITEM FROM HACKERNEWS DESERIALIIZED TO ITEM
func fetch(kid int, itemsUri string) (*Item, error) {
	var childMap Item
	res, err := http.Get(fmt.Sprintf(itemsUri, kid))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	byteData, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	err = json.Unmarshal(byteData, &childMap)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	childMap.FetchedAt = time.Now()
	return &childMap, nil
}
