package logviewer

import (
	"errors"
	"log"
	"strings"

	"github.com/fluent/fluent-logger-golang/fluent"
)

type DataType int

const (
	Task DataType = iota
	Job
)

// LogHandler Corrdinator,TaskRunner用のログハンドラ
// ログビューアサービス用にハンドリングする。
// ログはfluentdで送信する
//
// 使用例
// loghandler, err := logviewer.NewTaskLogHandler(logviewer.Task, fluent.Config{})
// if err != nil {
// 	panic(err)
// }
// runner := gojobcoordinatortest.NewTaskRunner(gojobcoordinatortest.TaskRunnerConfig{TaskNumMax: 2, Handler: &loghandler})
type LogHandler struct {
	dataType        DataType
	logTag          string
	startTag        string
	startLogPattern string
	logger          *fluent.Fluent
}

// Close 終了処理。fluentdの接続解除を行う。
func (l *LogHandler) Close() {
	l.logger.Close()
}

// HandleLog gojobcoordinatortest.LogHandlerインターフェイスの実装
// 受け取ったログをfulentdに送る
func (l *LogHandler) HandleLog(id string, p []byte) {
	logStr := string(p)

	if strings.Contains(logStr, l.startLogPattern) {
		if err := l.logger.Post(l.startTag, map[string]string{"id": id}); err != nil {
			log.Printf("Warning: ログ送信に失敗しました。: %s", err.Error())
		}
	}

	if err := l.logger.Post(l.logTag, map[string]string{"id": id, "log": logStr}); err != nil {
		log.Printf("Warning: ログ送信に失敗しました。: %s", err.Error())
	}
}

// NewTaskLogHandler ログハンドラの作成
// dataTypeにログの種類を指定。タスク(TaskRunner向け)かジョブ(Corrdinator向け)かを指定する。
func NewTaskLogHandler(dataType DataType, fluentConf fluent.Config) (LogHandler, error) {
	logger, err := fluent.New(fluentConf)
	if err != nil {
		return LogHandler{}, err
	}

	var logTag, startTag, startLogPattern string
	switch dataType {
	case Task:
		logTag = "logViewer.task"
		startTag = "logViewer.taskStart"
		startLogPattern = "Start Task."
	case Job:
		logTag = "logViewer.job"
		startTag = "logViewer.jobStart"
		startLogPattern = "Start Job."
	default:
		return LogHandler{}, errors.New("ハンドラータイプが不正です")
	}

	return LogHandler{dataType: dataType, logger: logger, logTag: logTag, startTag: startTag, startLogPattern: startLogPattern}, nil
}
