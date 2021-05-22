package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

// coordinatorServer TaskRunnerを管理してタスクを振り分けるサーバー
type coordinatorServer struct {
	jobs        sync.Map
	runnerAddrs sync.Map
}

func (coordinator *coordinatorServer) newRouter() *mux.Router {
	r := mux.NewRouter()

	// ジョブスタート
	r.HandleFunc("/start", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var startReq gojobcoordinatortest.JobStartRequest
		if !gojobcoordinatortest.ReadJSONFromRequest(rw, r, &startReq) {
			return
		}

		jobID, err := coordinator.newJob()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		job, err := coordinator.getJob(jobID)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		go job.Run(jobID, coordinator, &startReq)

		response := gojobcoordinatortest.JobStartResponse{ID: jobID}
		err = json.NewEncoder(rw).Encode(response)
		if err != nil {
			http.Error(rw, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		}

	}).Methods("POST")

	// ジョブキャンセル
	r.HandleFunc("/cancel/{jobID}", func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		job, err := coordinator.getJob(vars["jobID"])
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		job.Cancel()
	}).Methods("POST")

	// ジョブステータス取得
	r.HandleFunc("/status/{jobID}", func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		job, err := coordinator.getJob(vars["jobID"])
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		status := job.GetStatus()
		err = json.NewEncoder(rw).Encode(status)
		if err != nil {
			http.Error(rw, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		}

	}).Methods("GET")

	// TaskRunner接続
	r.HandleFunc("/connect", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var connectionReq gojobcoordinatortest.TaskRunnerConnectionRequest
		if !gojobcoordinatortest.ReadJSONFromRequest(rw, r, &connectionReq) {
			return
		}

		err := coordinator.connectRunner(connectionReq.Address)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

	}).Methods("POST")

	// TaskRunner接続解除
	r.HandleFunc("/disconnect", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var connectionReq gojobcoordinatortest.TaskRunnerConnectionRequest
		if !gojobcoordinatortest.ReadJSONFromRequest(rw, r, &connectionReq) {
			return
		}

		err := coordinator.disconnectRunner(connectionReq.Address)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

	}).Methods("POST")

	// 接続しているRunner取得
	r.HandleFunc("/list", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var runners []string
		addRunner := func(addr, _ interface{}) bool {
			runners = append(runners, addr.(string))
			return true
		}
		coordinator.runnerAddrs.Range(addRunner)
		responseData := gojobcoordinatortest.RunnerListResponse{Runners: runners}
		err := json.NewEncoder(rw).Encode(responseData)
		if err != nil {
			http.Error(rw, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		}
	}).Methods("GET")

	return r
}

func (coordinator *coordinatorServer) startTask(req *gojobcoordinatortest.TaskStartRequest) (string, string, error) {
	var returnAddr, returnID string
	taskStarted := false

	startFunc := func(addr, _ interface{}) bool {
		id, err := requestStartTask(addr.(string), req)
		if err == nil {
			returnAddr = addr.(string)
			returnID = id
			taskStarted = true
			return false
		}

		return true
	}

	coordinator.runnerAddrs.Range(startFunc)

	if taskStarted {
		return returnAddr, returnID, nil
	}

	return "", "", errors.New("タスクを開始出来ませんでした")
}

// requestStartTask 指定したTaskRunnerサーバーにタスク開始をリクエストする
func requestStartTask(runnerAddr string, req *gojobcoordinatortest.TaskStartRequest) (string, error) {
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

	var startResponse gojobcoordinatortest.TaskStartResponse
	err = gojobcoordinatortest.ReadJSONFromResponse(res, &startResponse)
	if err != nil {
		return "", err
	}

	return startResponse.ID, nil
}

func (coordinator *coordinatorServer) connectRunner(runnerAddr string) error {
	_, exist := coordinator.runnerAddrs.Load(runnerAddr)
	if exist == true {
		return errors.New(fmt.Sprint("すでに接続済みです:", runnerAddr))
	}

	log.Println("TaskRunnerを接続しました:", runnerAddr)
	coordinator.runnerAddrs.Store(runnerAddr, nil)

	return nil
}

func (coordinator *coordinatorServer) disconnectRunner(runnerAddr string) error {
	_, exist := coordinator.runnerAddrs.Load(runnerAddr)
	if exist == false {
		return errors.New(fmt.Sprint("接続されていません:", runnerAddr))
	}

	coordinator.runnerAddrs.Delete(runnerAddr)

	log.Println("TaskRunnerを切断しました:", runnerAddr)

	return nil
}

func (coordinator *coordinatorServer) newJob() (string, error) {

	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	_, exist := coordinator.jobs.Load(id.String())
	if exist == true {
		return "", errors.New("ID重複")
	}

	coordinator.jobs.Store(id.String(), &job{})

	return id.String(), nil
}

func (coordinator *coordinatorServer) getJob(jobID string) (*job, error) {
	value, ok := coordinator.jobs.Load(jobID)
	if !ok {
		return nil, fmt.Errorf("ジョブID %v は存在しません", jobID)
	}

	job, ok := value.(*job)
	if !ok {
		return nil, fmt.Errorf("ジョブID %v の取得に失敗しました", jobID)
	}

	return job, nil
}
