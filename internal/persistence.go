package internal

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"os"
)

const (
	tmpname      = "data.tmp"
	filename     = "data.json"
	gziptmpname  = "data.tmp.gz"
	gzipfilename = "data.json.gz"
)

type Persistencer interface {
	Store(data HNData) error
	Fetch() (HNData, error)
}

type jsonToFs struct {
}
type gzipJsonToFs struct {
}

func NewGzipFsPeristence() Persistencer {
	return &gzipJsonToFs{}
}

func (j *gzipJsonToFs) Store(data HNData) error {
	res, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return err
	}
	fbuf, err := os.Create(gziptmpname)
	defer fbuf.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	zw := gzip.NewWriter(fbuf)
	_, err = zw.Write(res)
	defer zw.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	os.Rename(gziptmpname, gzipfilename)
	return nil
}
func (j *gzipJsonToFs) Fetch() (HNData, error) {
	bs, err := os.Open(gzipfilename)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	bufReader := bufio.NewReader(bs)
	gzReader, err := gzip.NewReader(bufReader)
	defer gzReader.Close()
	if err != nil {
		log.Fatal(err)
	}
	data, _ := io.ReadAll(gzReader)

	var res HNData
	err = json.Unmarshal(data, &res)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return res, nil
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
	err = os.WriteFile(tmpname, res, 0777)
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
