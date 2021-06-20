package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

type taskInfo struct {
	id          string
	runnderAddr string
}

type job struct {
	taskInfos     []taskInfo
	taskInfosLock sync.Mutex
	cancelFunc    context.CancelFunc
	busy          bool
	id            string
	logger        log.Logger
}

func NewJob(jobID string, logger log.Logger) *job {
	return &job{id: jobID, logger: logger}
}

func (j *job) Run(coordinator *coordinatorServer, jobReq *gojobcoordinatortest.JobStartRequest) {
	j.busy = true
	j.logger.Print("Start Job.")

	var ctx context.Context
	ctx, j.cancelFunc = context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < len(jobReq.Tasks); i++ {
		wg.Add(1)
		go j.runTask(ctx, &wg, coordinator, &jobReq.Tasks[i], jobReq.TargetFilters)
	}
	wg.Wait()

	j.busy = false
	j.logger.Print("Complete Job.")
}

func (j *job) runTask(ctx context.Context, wg *sync.WaitGroup, coordinator *coordinatorServer, taskReq *gojobcoordinatortest.TaskStartRequest, targets *[]string) {
	defer wg.Done()

	// タスク開始成功するまで繰り返す
	var runnerAddr, taskID string
	ticker := time.NewTicker(time.Second * 30)
	for {
		j.logger.Printf("タスク開始を試みます\n")

		var err error
		runnerAddr, taskID, err = coordinator.startTask(taskReq, targets)
		if err == nil {
			j.taskInfosLock.Lock()
			j.taskInfos = append(j.taskInfos, taskInfo{id: taskID, runnderAddr: runnerAddr})
			j.taskInfosLock.Unlock()
			j.logger.Printf("TaskRunner %v でタスクを開始しました %v\n", runnerAddr, taskID)
			break
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			// キャンセルされれば終了
			return
		}
	}

	// 開始成功したら完了するまで繰り返す
	for {
		status, err := getTaskStatus(runnerAddr, taskID)
		if err != nil {
			j.logger.Println(err)
			return
		}

		if status.Status != gojobcoordinatortest.StatusBusy {
			j.logger.Printf("TaskRunner %v で開始したTaskID %v が完了しました。", runnerAddr, taskID)
			return
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			// キャンセル指示があればキャンセルリクエストを投げる
			cancelRes, err := http.Post(fmt.Sprint(runnerAddr, "/cancel/", taskID), "", nil)
			if err != nil || cancelRes.StatusCode != http.StatusOK {
				j.logger.Printf("TaskRunner %v で開始したTaskID %v へのキャンセルに失敗しました。", runnerAddr, taskID)
				if err != nil {
					j.logger.Print(err)
				}
				return
			}
		}
	}
}

func getTaskStatus(runnerAddr, taskID string) (gojobcoordinatortest.TaskStatusResponse, error) {
	var result gojobcoordinatortest.TaskStatusResponse

	statusURL := fmt.Sprint(runnerAddr, "/status/", taskID)
	res, err := http.Get(statusURL)
	if err != nil {
		return result, fmt.Errorf("TaskRunner %v で開始したTaskID %v のステータス取得でエラーが発生しました。 %v", runnerAddr, taskID, err)
	}

	if res.StatusCode != http.StatusOK {
		return result, fmt.Errorf("TaskRunner %v で開始したTaskID %v のステータス取得でエラーが発生しました。 %v", runnerAddr, taskID, err)
	}

	err = gojobcoordinatortest.ReadJSONFromResponse(res, &result)
	res.Body.Close()
	if err != nil {
		return result, fmt.Errorf("TaskRunner %v で開始したTaskID %v のステータス解析でエラーが発生しました。 %v", runnerAddr, taskID, err)
	}

	return result, nil
}

func (j *job) Cancel() {
	if j.cancelFunc != nil {
		j.cancelFunc()
		log.Printf("[%v]ジョブのキャンセルリクエストを行ました", j.id)
	}
}

func (j *job) GetStatus() gojobcoordinatortest.JobStatusResponse {
	j.taskInfosLock.Lock()
	taskInfosCopy := make([]taskInfo, len(j.taskInfos))
	copy(taskInfosCopy, j.taskInfos)
	j.taskInfosLock.Unlock()

	response := gojobcoordinatortest.JobStatusResponse{}

	var statuses []gojobcoordinatortest.TaskStatusResponse

	for _, taskInfo := range taskInfosCopy {
		status, err := getTaskStatus(taskInfo.runnderAddr, taskInfo.id)
		if err != nil {
			log.Println(err)
		} else {
			statuses = append(statuses, status)
		}
	}

	response.Busy = j.busy
	response.TaskStatuses = &statuses

	return response
}
