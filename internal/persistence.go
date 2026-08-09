package internal

import (
	"encoding/json"
	"log"
	"os"
)

const (
	tmpname  = "data.tmp"
	filename = "data.json"
)

type Persistencer interface {
	Store(data HNData) error
	Fetch() (HNData, error)
}

type jsonToFs struct {
}

func NewFsPeristence() Persistencer {
	return &jsonToFs{}
}

func (j *jsonToFs) Store(data HNData) error {
	res, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return err
	}
	err = os.WriteFile(tmpname, res, 0700)
	if err != nil {
		log.Println(err)
		return err
	}
	os.Rename(tmpname, filename)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
func (j *jsonToFs) Fetch() (HNData, error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var res HNData
	err = json.Unmarshal(bs, &res)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return res, nil
}
