package logviewer

import (
	"errors"
	"log"
	"strings"

	"github.com/fluent/fluent-logger-golang/fluent"
)

type HandlerType int

const (
	Task HandlerType = iota
	Job
)

type LogHandler struct {
	handlerType     HandlerType
	logTag          string
	startTag        string
	startLogPattern string
	logger          *fluent.Fluent
}

func (l *LogHandler) Close() {
	l.logger.Close()
}

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

func NewTaskLogHandler(handlerType HandlerType, fluentConf fluent.Config) (LogHandler, error) {
	logger, err := fluent.New(fluentConf)
	if err != nil {
		return LogHandler{}, err
	}

	var logTag, startTag, startLogPattern string
	switch handlerType {
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

	return LogHandler{handlerType: handlerType, logger: logger, logTag: logTag, startTag: startTag, startLogPattern: startLogPattern}, nil
}
