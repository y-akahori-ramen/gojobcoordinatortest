package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

const (
	// ProcNameWait ウェイトを実行するタスク処理名
	ProcNameWait string = "Wait"

	// ProcNameEcho 受け取った文字列を出力するタスク処理名
	ProcNameEcho string = "Echo"
)

// 指定された値を出力するタスク
type taskEcho struct {
	value string
}

func (task *taskEcho) Run(ctx context.Context, taskID string, logger *log.Logger, done chan<- *gojobcoordinatortest.TaskResult) {
	logger.Println("Echo:", task.value)
	done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: true}
}

func newEchoTask(req *gojobcoordinatortest.TaskStartRequest) (gojobcoordinatortest.Task, error) {
	if req.ProcName != ProcNameEcho {
		return nil, fmt.Errorf("処理名 %s が不正です", req.ProcName)
	}

	if req.Params == nil {
		return nil, fmt.Errorf("%sに必要なパラメータValueが存在しません", req.ProcName)
	}

	value, ok := (*req.Params)[string("Value")]

	if !ok {
		return nil, fmt.Errorf("%sに必要なパラメータValueが存在しません", req.ProcName)
	}

	valueStr, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("%sのパラメータValueは文字列で指定してください", req.ProcName)
	}

	return &taskEcho{value: valueStr}, nil
}

// 指定された時間待機するタスク
type taskWait struct {
	waitSec float64
}

func (task *taskWait) Run(ctx context.Context, taskID string, logger *log.Logger, done chan<- *gojobcoordinatortest.TaskResult) {
	duration := time.Duration(task.waitSec * (float64)(time.Second))
	ticker := time.NewTicker(duration)
	logger.Printf("[taskWait][%v]%v待機します", taskID, duration.String())
	defer ticker.Stop()

	select {
	case <-ticker.C:
		logger.Printf("[taskWait][%v]待機が完了しました", taskID)
		done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: true}
		return
	case <-ctx.Done():
		logger.Printf("[taskWait][%v]キャンセルが発生しました", taskID)
		done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: false}
		return
	}
}

func newWaitTask(req *gojobcoordinatortest.TaskStartRequest) (gojobcoordinatortest.Task, error) {
	if req.ProcName != ProcNameWait {
		return nil, fmt.Errorf("処理名 %s が不正です", req.ProcName)
	}

	if req.Params == nil {
		return nil, fmt.Errorf("%sに必要なパラメータSecが存在しません", req.ProcName)
	}

	value, ok := (*req.Params)[string("Sec")]
	if !ok {
		return nil, fmt.Errorf("%sに必要なパラメータSecが存在しません", req.ProcName)
	}

	sec, ok := value.(float64)
	if !ok {
		return nil, fmt.Errorf("%sのパラメータSecは数値で指定してください", req.ProcName)
	}

	return &taskWait{waitSec: sec}, nil
}
