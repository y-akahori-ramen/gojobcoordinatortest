package gojobcoordinatortest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type taskInfo struct {
	id          string
	runnderAddr string
}

type coordinatorJob struct {
	taskInfos     []taskInfo
	taskInfosLock sync.Mutex
	cancelFunc    context.CancelFunc
	busy          bool
	id            string
	logger        log.Logger
}

func newCoordinatorJob(jobID string, logger log.Logger) *coordinatorJob {
	return &coordinatorJob{id: jobID, logger: logger}
}

func (j *coordinatorJob) run(cod *Coordinator, jobReq *JobStartRequest) {
	j.busy = true
	j.logger.Print("Start Job.")

	var ctx context.Context
	ctx, j.cancelFunc = context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < len(jobReq.Tasks); i++ {
		wg.Add(1)
		go j.runTask(ctx, &wg, cod, &jobReq.Tasks[i], jobReq.TargetFilters)
	}
	wg.Wait()

	j.busy = false
	j.logger.Print("Complete Job.")
}

func (j *coordinatorJob) runTask(ctx context.Context, wg *sync.WaitGroup, cod *Coordinator, taskReq *TaskStartRequest, targets *[]string) {
	defer wg.Done()

	// タスク開始成功するまで繰り返す
	var runnerAddr, taskID string
	ticker := time.NewTicker(time.Second * 30)
	for {
		j.logger.Printf("タスク開始を試みます\n")

		var err error
		runnerAddr, taskID, err = cod.startTask(taskReq, targets)
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

		if status.Status != StatusBusy {
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

func getTaskStatus(runnerAddr, taskID string) (TaskStatusResponse, error) {
	var result TaskStatusResponse

	statusURL := fmt.Sprint(runnerAddr, "/status/", taskID)
	res, err := http.Get(statusURL)
	if err != nil {
		return result, fmt.Errorf("TaskRunner %v で開始したTaskID %v のステータス取得でエラーが発生しました。 %v", runnerAddr, taskID, err)
	}

	if res.StatusCode != http.StatusOK {
		return result, fmt.Errorf("TaskRunner %v で開始したTaskID %v のステータス取得でエラーが発生しました。 %v", runnerAddr, taskID, err)
	}

	err = ReadJSONFromResponse(res, &result)
	res.Body.Close()
	if err != nil {
		return result, fmt.Errorf("TaskRunner %v で開始したTaskID %v のステータス解析でエラーが発生しました。 %v", runnerAddr, taskID, err)
	}

	return result, nil
}

func (j *coordinatorJob) cancel() {
	if j.cancelFunc != nil {
		j.cancelFunc()
		log.Printf("[%v]ジョブのキャンセルリクエストを行ました", j.id)
	}
}

func (j *coordinatorJob) getStatus() JobStatusResponse {
	j.taskInfosLock.Lock()
	taskInfosCopy := make([]taskInfo, len(j.taskInfos))
	copy(taskInfosCopy, j.taskInfos)
	j.taskInfosLock.Unlock()

	response := JobStatusResponse{}

	var statuses []TaskStatusResponse

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
