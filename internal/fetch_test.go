package internal

import (
	"encoding/json"
	"sync"
	"testing"
)

const (
	RESULT = `[{"id":1000,"score":4,"time":1172394646,"title":"How Important is the .com TLD?","text":"","by":"python_kiss","kids":null,"url":"http://www.netbusinessblog.com/2007/02/19/how-important-is-the-dot-com/","parent":0,"dead":false,"Comments":null}]`

	TEST_TWO_RESULT = `[{"id":80,"score":0,"time":1171860682,"title":"","text":"yet another advantage of being in the Bay Area","by":"pg","kids":null,"url":"","parent":79,"dead":false,"Comments":null},{"id":900,"score":0,"time":1172343574,"title":"","text":"I've listened to most of these a few months back and they're great. The tom Coates one contains brilliant advice for anyone creating a web app. His slides are available at http://www.plasticbag.org/archives/2006/02/my_future_of_web_apps_slides/\u003cp\u003eThis page will be updated within the next few weeks with audio from the latest FOWA conference.","by":"danw","kids":[949],"url":"","parent":833,"dead":false,"Comments":[{"id":949,"score":0,"time":1172366912,"title":"","text":"Thanks, I hadn't seen those slides, and they're very good.","by":"phil","kids":null,"url":"","parent":900,"dead":false,"Comments":null}]},{"id":85,"score":5,"time":1171906738,"title":"Hack your desk to remove the clutter","text":"","by":"nate","kids":[85945],"url":"http://lifehacker.com/software/diy/diy-underdesk-gadget-mount-237789.php","parent":0,"dead":false,"Comments":[{"id":85945,"score":0,"time":1196780254,"title":"","text":"","by":"","kids":null,"url":"","parent":85,"dead":false,"Comments":null}]}]`
)

func TestFetchArticleIDs(t *testing.T) {
	//Takes CONST RESULT
	f := NewFetcher(1)
	result, _ := f.FetchItemsConcurrentlyByIDs(1000)
	json, _ := json.Marshal(result)
	if string(json) != RESULT {
		t.Errorf("Fetched Items are not the same. Should be %s \n\n but is \n\n %s\n", RESULT, string(json))
	}
}

func TestFetchArticleIDsUnequaly(t *testing.T) {
	//Takes CONST RESULT
	f := NewFetcher(1)
	result, _ := f.FetchItemsConcurrentlyByIDs(999)
	json, _ := json.Marshal(result)
	if string(json) == RESULT {
		t.Errorf("Fetched Items are not the same. Should be %s \n\n but is \n\n %s\n", RESULT, string(json))
	}
}

func TestFetchManyArticles(t *testing.T) {
	//Takes CONST TEST_TWO_RESULT
	f := NewFetcher(1)
	result, _ := f.FetchItemsConcurrentlyByIDs(80, 85, 900)
	json, _ := json.Marshal(result)
	if string(json) == TEST_TWO_RESULT {
		t.Errorf("Fetched Items are not the same. Should be %s \n\n but is \n\n %s\n", RESULT, string(json))
	}
}

func TestGetLatestArticle(t *testing.T) {
	var wg sync.WaitGroup
	f := NewFetcher(1)
	f.CommentDownload(false)
	latestId, err := f.NewestArticleID()
	if err != nil {
		t.Errorf("Could not get latest Hackernews ID\n\n %s \n", err)
	}
	var result []*Item
	wg.Add(2)
	go func() {
		defer wg.Done()
		result, err = f.FetchItemsConcurrentlyByIDs(latestId)
		if err != nil {
			t.Errorf("Could not get latest Hackernews ID\n\n %s \n", err)
		}
	}()
	var result2 Item
	go func() {
		defer wg.Done()
		result2, err = f.GetLatestArticle()
		if err != nil {
			t.Errorf("Could not get latest Hackernews ID\n\n %s \n", err)
		}
	}()
	wg.Wait()
	if len(result) == 0 {
		t.Errorf("Should have an result")
	}
	resultFoo := result[0]

	if resultFoo.Id != result2.Id {
		t.Errorf("Latest Articles have to be the same. Should %d \n\n Got %d \n\n", resultFoo.Id, result2.Id)
	}
}
