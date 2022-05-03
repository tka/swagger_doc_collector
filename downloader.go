package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type Downloader struct {
	Docs   []Doc
	ticker *time.Ticker
	quit   chan bool
}

func (downloader *Downloader) Start() {
	downloader.ticker = time.NewTicker(time.Second * time.Duration(config.Interval))
	downloader.quit = make(chan bool)
	downloader.Docs = config.Docs
	logger.Info("Downloader started")
	downloader.downloadDocs()
	go func() {
		for {
			select {
			case <-downloader.ticker.C:
				downloader.downloadDocs()
			case <-downloader.quit:
				downloader.ticker.Stop()
				return
			}
		}
	}()

}

func (downloader *Downloader) Stop() {
	downloader.quit <- true
	logger.Info("Downloader stopped")
}

func (downloader *Downloader) downloadDocs() {
	for index := range downloader.Docs {
		go func(index int) {
			downloader.downloadFile(config.Docs[index])
		}(index)
	}

}

func (downloader *Downloader) downloadFile(doc Doc) {
	logger.Info("Downloading", zap.Any("doc", doc))
	now := time.Now().Unix()
	filepath := path.Join(doc.Path(), strconv.FormatInt(now, 10))

	url := doc.Url
	folder := doc.Path()
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		os.MkdirAll(folder, os.ModePerm)
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("download file fail", zap.String("url", url), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		logger.Error("download file status error", zap.String("url", url), zap.Int("status", resp.StatusCode))
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("read body error ", zap.String("url", url), zap.Error(err))
		return
	}
	logger.Info(url)
	isSame, err := isSameLeastVersion(folder, body)
	if isSame {
		logger.Info("same version", zap.String("url", url))
		return
	} else if err != nil {
		// log and create new version
		logger.Error("check api diff error", zap.String("url", url), zap.Error(err))
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		logger.Error("create file path fail", zap.String("path", filepath), zap.Error(err))
		return
	}
	defer out.Close()
	// Writer the body to file
	_, err = out.Write(body)
	if err != nil {
		logger.Error("write file error", zap.String("url", url), zap.String("path", filepath), zap.Error(err))
		return
	}

}
