package main

import (
	"context"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

// TaskResult タスク処理結果
type TaskResult struct {
	ID           string
	Success      bool
	ResultValues *map[string]interface{}
}

// Task タスクインターフェイス
type Task interface {
	Run(ctx context.Context, taskID string, done chan<- *TaskResult)
}

// TaskFactoryFunc タスク生成関数の型
type TaskFactoryFunc func(req *gojobcoordinatortest.TaskStartRequest) (Task, error)
