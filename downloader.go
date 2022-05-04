package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/tufin/oasdiff/diff"
	"github.com/tufin/oasdiff/report"
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

	isSame, diff, err := isSameLeastVersion(folder, body)
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

	SendSlackNotification(doc, diff)
}

type SlackRequestBody struct {
	Text string `json:"text"`
}

// SendSlackNotification will post to an 'Incoming Webook' url setup in Slack Apps. It accepts
// some text and the slack channel is saved within Slack.
func SendSlackNotification(doc Doc, diff *diff.Diff) error {
	msg := doc.Name + " 串接文件有更新\n```\n" + report.GetTextReportAsString(diff) + "\n```"
	slackBody, _ := json.Marshal(SlackRequestBody{Text: msg})
	req, err := http.NewRequest(http.MethodPost, config.SlackWebhookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return errors.New("Non-ok response returned from Slack")
	}
	return nil
}
