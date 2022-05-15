package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-swagger/go-swagger/cmd/swagger/commands/diff"

	"github.com/go-openapi/loads"
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

func (downloader *Downloader) downloadFile(doc Doc) (err error) {
	logger.Info("Downloading", zap.Any("doc", doc))
	defer func() {
		/*
			if x := recover(); x != nil {
				sendSlackMessage("串接文件 " + doc.Name + " 下载失敗")
				logger.Error("Download Failed", zap.Any("doc", doc), zap.Any("err", x))
			}
			//*/
		if err != nil {
			sendSlackMessage("串接文件 " + doc.Name + " 下载失敗")
			logger.Error("Download Failed", zap.Any("doc", doc), zap.Error(err))
		}
	}()

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
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		logger.Error("download file status error", zap.String("url", url), zap.Int("status", resp.StatusCode))
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("read body error ", zap.String("url", url), zap.Error(err))
		return err
	}
	for i := 0; i < 1; i++ {
		jsonRaw := json.RawMessage(body)
		newSpec, err := loads.Analyzed(jsonRaw, "")
		if err != nil {
			logger.Error("analyzed json raw error", zap.String("url", url), zap.Error(err))
			return err
		}
		/*
			opt := spec.ExpandOptions{
				AbsoluteCircularRef: false,
			}
			expandedDoc, err := newSpec.Expanded(&opt)
			if err != nil {
				logger.Error("expanded doc error", zap.String("url", url), zap.Error(err))
				return err
			}
			expandedDocByte, err := json.MarshalIndent(expandedDoc.Spec(), "", "  ")
			if err != nil {
				logger.Error("marshal expanded doc error", zap.String("url", url), zap.Error(err))
				return err
			}
			logger.Info("index", zap.Int("i", i), zap.Int("body", len(body)), zap.Int("len", len(expandedDocByte)))
			//continue
			// remove recursion $ref in doc
			r, _ := regexp.Compile(`\$ref[^}]*`)
			finalDoc := r.ReplaceAllString(string(expandedDocByte), "type\":\"object\"")
			expandedDocByte = []byte(finalDoc)
			newSpec, err = loads.Analyzed(expandedDocByte, "")
			if err != nil {
				return err
			}*/
		isSame, diffs, err := isSameLeastVersion(folder, newSpec)
		if isSame {
			logger.Info("same version", zap.String("url", url))
			return nil
		} else if err != nil {
			// log and create new version
			logger.Error("check api diff error", zap.String("url", url), zap.Error(err))
			return err

		}

		// Create the file
		out, err := os.Create(filepath)
		if err != nil {
			logger.Error("create file path fail", zap.String("path", filepath), zap.Error(err))
			return err
		}
		defer out.Close()
		newSpecByte, err := json.MarshalIndent(newSpec.Spec(), "", "  ")
		if err != nil {
			logger.Error("marshal newSpec doc error", zap.String("url", url), zap.Error(err))
			return err
		}
		// Writer the body to file
		_, err = out.Write(newSpecByte)
		if err != nil {
			logger.Error("write file error", zap.String("url", url), zap.String("path", filepath), zap.Error(err))
			return err
		} else {
			logger.Info("write file success", zap.String("url", url), zap.String("path", filepath))
		}

		return SendSlackNotification(doc, diffs)
	}
	return nil
}

type SlackRequestBody struct {
	Text string `json:"text"`
}

// SendSlackNotification will post to an 'Incoming Webook' url setup in Slack Apps. It accepts
// some text and the slack channel is saved within Slack.

func SendSlackNotification(doc Doc, diffs *diff.SpecDifferences) error {
	if diffs == nil {
		return nil
	}
	allDiff, err, _ := diffs.ReportAllDiffs(false)
	if err != nil {
		return err
	}
	diffStringBuider := new(strings.Builder)
	_, err = io.Copy(diffStringBuider, allDiff)
	if err != nil {
		return err
	}

	msg := doc.Name + " 串接文件有更新\n```\n" + diffStringBuider.String() + "\n```"
	return sendSlackMessage(msg)
}
func sendSlackMessage(msg string) error {
	if config.SlackWebhookUrl == "" {
		return nil
	}
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
