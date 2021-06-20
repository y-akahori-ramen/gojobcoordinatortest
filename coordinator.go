package gojobcoordinatortest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CoordinatorConfig コーディネータの設定項目
// Handler ジョブのログ出力ハンドリング。不要な場合はnilを指定する。
type CoordinatorConfig struct {
	Handler LogHandler
}

// Coordinator TaskRunnerServerを管理してタスクを振り分ける
type Coordinator struct {
	CoordinatorConfig
	jobs        sync.Map
	runnerAddrs sync.Map
}

// NewCoordinator Coordinatorの作成
func NewCoordinator(config CoordinatorConfig) *Coordinator {
	return &Coordinator{CoordinatorConfig: config}
}

// Run Coordinatorの起動
func (cod *Coordinator) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-ticker.C:
			cod.removeDeadTaskRunners()
		case <-ctx.Done():
			return
		}
	}
}

func (cod *Coordinator) Start(req JobStartRequest) (JobStartResponse, error) {
	resp := JobStartResponse{}
	jobID, err := cod.newJob()
	if err != nil {
		return resp, err
	}

	job, err := cod.getJob(jobID)
	if err != nil {
		return resp, err
	}

	go job.run(cod, &req)

	resp.ID = jobID
	return resp, err
}

func (cod *Coordinator) Cancel(id string) error {
	job, err := cod.getJob(id)
	if err != nil {
		return err
	}

	job.cancel()

	return nil
}

func (cod *Coordinator) GetStatus(id string) (JobStatusResponse, error) {
	job, err := cod.getJob(id)
	if err != nil {
		return JobStatusResponse{}, err
	}

	return job.getStatus(), err
}

func (cod *Coordinator) Connect(req TaskRunnerConnectionRequest) error {
	_, exist := cod.runnerAddrs.Load(req.Address)
	if exist == true {
		return errors.New(fmt.Sprint("すでに接続済みです:", req.Address))
	}

	log.Println("TaskRunnerを接続しました:", req.Address)
	cod.runnerAddrs.Store(req.Address, nil)

	return nil
}

func (cod *Coordinator) Disconnect(req TaskRunnerConnectionRequest) error {
	_, exist := cod.runnerAddrs.Load(req.Address)
	if exist == false {
		return errors.New(fmt.Sprint("接続されていません:", req.Address))
	}

	cod.runnerAddrs.Delete(req.Address)

	log.Println("TaskRunnerを切断しました:", req.Address)

	return nil
}

func (cod *Coordinator) GetRunners() RunnerListResponse {
	var runners []string
	addRunner := func(addr, _ interface{}) bool {
		runners = append(runners, addr.(string))
		return true
	}
	cod.runnerAddrs.Range(addRunner)
	return RunnerListResponse{Runners: runners}
}

func (cod *Coordinator) GetJobs() JobListResponse {
	var jobs []string
	addJob := func(job, _ interface{}) bool {
		jobs = append(jobs, job.(string))
		return true
	}
	cod.jobs.Range(addJob)
	return JobListResponse{Jobs: jobs}
}

func (cod *Coordinator) startTask(req *TaskStartRequest, targets *[]string) (string, string, error) {
	var returnAddr, returnID string
	taskStarted := false

	startFunc := func(addr, _ interface{}) bool {
		addrStr := addr.(string)

		// 対象の指定がある場合は有効な対象かをチェック。対象外であればタスク開始は行わない。
		if targets != nil {
			isValidTarget := false
			for _, target := range *targets {
				if strings.Contains(addrStr, target) {
					isValidTarget = true
					break
				}
			}
			if !isValidTarget {
				return true
			}
		}

		id, err := requestStartTask(addrStr, req)
		if err == nil {
			returnAddr = addrStr
			returnID = id
			taskStarted = true
			return false
		}

		return true
	}

	cod.runnerAddrs.Range(startFunc)

	if taskStarted {
		return returnAddr, returnID, nil
	}

	return "", "", errors.New("タスクを開始出来ませんでした")
}

// requestStartTask 指定したTaskRunnerサーバーにタスク開始をリクエストする
func requestStartTask(runnerAddr string, req *TaskStartRequest) (string, error) {
	url := fmt.Sprint(runnerAddr, "/start")
	json, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	res, err := http.Post(url, "application/json", bytes.NewBuffer(json))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", errors.New("タスク開始に失敗しました")
	}

	var startResponse TaskStartResponse
	err = ReadJSONFromResponse(res, &startResponse)
	if err != nil {
		return "", err
	}

	return startResponse.ID, nil
}

func (cod *Coordinator) newJob() (string, error) {

	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	_, exist := cod.jobs.Load(id.String())
	if exist == true {
		return "", errors.New("ID重複")
	}

	writer := &WriterWithHandler{Writer: log.Default().Writer(), Handler: cod.Handler, Id: id.String()}
	logger := log.New(writer, fmt.Sprintf("[%s]", id.String()), log.Default().Flags())
	cod.jobs.Store(id.String(), newCoordinatorJob(id.String(), *logger))

	return id.String(), nil
}

func (cod *Coordinator) getJob(jobID string) (*coordinatorJob, error) {
	value, ok := cod.jobs.Load(jobID)
	if !ok {
		return nil, fmt.Errorf("ジョブID %v は存在しません", jobID)
	}

	job, ok := value.(*coordinatorJob)
	if !ok {
		return nil, fmt.Errorf("ジョブID %v の取得に失敗しました", jobID)
	}

	return job, nil
}

// checkAliveTaskRunners 接続しているTaskRunnerが生存しているかを確認し、生存していなければ接続リストから削除する
func (cod *Coordinator) removeDeadTaskRunners() {
	var wg sync.WaitGroup
	for _, runnerAddr := range cod.GetRunners().Runners {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			url := fmt.Sprint(addr, "/alive")
			resp, err := http.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				log.Println("TaskRunnerが生存していません:", addr)
				cod.Disconnect(TaskRunnerConnectionRequest{Address: addr})
			}
		}(runnerAddr)
	}
	wg.Wait()
}
