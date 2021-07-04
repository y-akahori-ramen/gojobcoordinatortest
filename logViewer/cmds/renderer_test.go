package main

import (
	"fmt"
	"os"
	"testing"

	logviewer "github.com/y-akahori-ramen/gojobcoordinatortest/logViewer"
)

type logDataMock struct {
	logData map[string]string
}

func newLogDataMock() *logDataMock {
	logData := map[string]string{}
	idList := []string{"aaaa", "bbbb", "cccc"}
	for _, id := range idList {
		logText := ""
		for i := 0; i < 10; i++ {
			logText += fmt.Sprintf("text:%d\n", i)
		}
		logData[id] = logText
	}
	return &logDataMock{logData: logData}
}

// GetTaskLog 指定したタスクIDのログを時間の昇順で取得します
func (data *logDataMock) GetTaskLog(taskID string) (string, error) {
	return data.getLog(taskID)
}

// GetJobLog 指定したジョブIDのログを時間の昇順で取得します
func (data *logDataMock) GetJobLog(jobID string) (string, error) {
	return data.getLog(jobID)
}

// GetTaskList タスクID一覧を取得します
func (data *logDataMock) GetTaskList() ([]logviewer.Summary, error) {
	return data.getList()
}

// GetJobList ジョブID一覧を取得します
func (data *logDataMock) GetJobList() ([]logviewer.Summary, error) {
	return data.getList()
}

func (data *logDataMock) getLog(id string) (string, error) {
	log, ok := data.logData[id]
	if ok {
		return log, nil
	} else {
		return "", fmt.Errorf("ログが存在しません: %s", id)
	}
}

func (data *logDataMock) getList() ([]logviewer.Summary, error) {
	var idList []logviewer.Summary
	for key := range data.logData {
		idList = append(idList, logviewer.Summary{Id: key, FirstLog: "FirstLog:" + key})
	}
	return idList, nil
}

func TestRenderer(t *testing.T) {
	renderer, err := NewRenderer(newLogDataMock())
	if err != nil {
		t.Fatal(err)
	}
	err = renderer.RenderIndex(os.Stdout)
	if err != nil {
		t.Fatal(err)
	}
	err = renderer.RenderList(os.Stdout, logviewer.Job)
	if err != nil {
		t.Fatal(err)
	}
	err = renderer.RenderList(os.Stdout, logviewer.Task)
	if err != nil {
		t.Fatal(err)
	}
	err = renderer.RenderLog(os.Stdout, logviewer.Task, "aaaa")
	if err != nil {
		t.Fatal(err)
	}
	err = renderer.RenderLog(os.Stdout, logviewer.Job, "aaaa")
	if err != nil {
		t.Fatal(err)
	}
}
