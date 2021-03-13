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

type taskResult struct {
	ID           string
	Success      bool
	ResultValues *map[string]interface{}
}

// Runにはキャンセル用のctxと完了時に結果を送ってもらうdoneを渡す
type task interface {
	Run(ctx context.Context, taskID string, done chan<- *taskResult)
}

// 指定された値を出力するタスク
type taskEcho struct {
	value string
}

func (task *taskEcho) Run(ctx context.Context, taskID string, done chan<- *taskResult) {
	log.Println("Echo:", task.value)
	done <- &taskResult{ID: taskID, Success: true}
}

// 指定された時間待機するタスク
type taskWait struct {
	waitSec float64
}

func (task *taskWait) Run(ctx context.Context, taskID string, done chan<- *taskResult) {
	duration := time.Duration(task.waitSec * (float64)(time.Second))
	ticker := time.NewTicker(duration)
	log.Printf("[taskWait][%v]%v待機します", taskID, duration.String())
	defer ticker.Stop()

	select {
	case <-ticker.C:
		log.Printf("[taskWait][%v]待機が完了しました", taskID)
		done <- &taskResult{ID: taskID, Success: true}
		return
	case <-ctx.Done():
		log.Printf("[taskWait][%v]キャンセルが発生しました", taskID)
		done <- &taskResult{ID: taskID, Success: false}
		return
	}
}

func newTask(req *gojobcoordinatortest.TaskStartRequest) (task, error) {
	switch req.ProcName {
	case ProcNameWait:
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
	case ProcNameEcho:
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
	default:
		return nil, fmt.Errorf("処理名%sはサポート外です", req.ProcName)
	}
}
