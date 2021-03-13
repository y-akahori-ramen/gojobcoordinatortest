package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

type taskStatus struct {
	result  *taskResult
	cancel  context.CancelFunc
	reqData gojobcoordinatortest.TaskStartRequest
}

type taskRunnerServer struct {
	resultDone       chan *taskResult
	taskStatuses     map[string]*taskStatus
	taskStatusesLock sync.Mutex
}

func (server *taskRunnerServer) handleTaskStart(w http.ResponseWriter, r *http.Request) {
	server.taskStatusesLock.Lock()
	defer server.taskStatusesLock.Unlock()

	var requestData gojobcoordinatortest.TaskStartRequest
	ok := gojobcoordinatortest.ReadJSONFromRequest(w, r, &requestData)
	if !ok {
		return
	}

	// タスク作成
	task, err := newTask(&requestData)
	if err != nil {
		http.Error(w, fmt.Sprint("タスク作成に失敗しました:", err.Error()), http.StatusBadRequest)
		return
	}

	// タスクID割り振り
	id, err := uuid.NewRandom()
	if err != nil {
		http.Error(w, fmt.Sprint("ID生成に失敗しました:", err.Error()), http.StatusInternalServerError)
		return
	}
	taskID := id.String()

	// タスクIDが確定するのでここでレスポンス作成　※タスク開始後にレスポンス作成失敗すると開始したタスクを止められないため
	result := gojobcoordinatortest.TaskStartResponse{TaskStartRequest: requestData, ID: taskID}
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, fmt.Sprint("タスク作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		return
	}

	// タスク状態管理情報の作成
	ctx, cancel := context.WithCancel(context.Background())
	server.taskStatuses[taskID] = &taskStatus{reqData: requestData, result: nil, cancel: cancel}

	// タスク実行
	// タスクが完了すればserver.resultDoneチャネルに結果が送られる
	// タスクをキャンセルする場合はserver.taskStatusesに保存しているキャンセル関数を呼ぶ
	go task.Run(ctx, taskID, server.resultDone)
	log.Printf("Start Task.　TaskID:%v ProcName:%v Params:%v\n", taskID, requestData.ProcName, requestData.Params)
}

func (server *taskRunnerServer) handleCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	server.taskStatusesLock.Lock()
	defer server.taskStatusesLock.Unlock()

	task, ok := server.taskStatuses[vars["taskID"]]
	if !ok {
		http.Error(w, fmt.Sprint("タスクIDが不正です:", vars["taskID"]), http.StatusNotFound)
		return
	}
	task.cancel()
}

func (server *taskRunnerServer) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	server.taskStatusesLock.Lock()
	defer server.taskStatusesLock.Unlock()

	task, ok := server.taskStatuses[vars["taskID"]]
	if !ok {
		http.Error(w, fmt.Sprint("タスクIDが不正です:", vars["taskID"]), http.StatusNotFound)
		return
	}

	var response gojobcoordinatortest.TaskStatusResponse
	response.TaskStartRequest = task.reqData
	if task.result != nil {
		if task.result.Success {
			response.Status = gojobcoordinatortest.StatusSuccess
		} else {
			response.Status = gojobcoordinatortest.StatusFailure
		}
		response.ResultValues = task.result.ResultValues
	} else {
		response.Status = gojobcoordinatortest.StatusBusy
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
	}
}

func (server *taskRunnerServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	server.taskStatusesLock.Lock()
	defer server.taskStatusesLock.Unlock()

	task, ok := server.taskStatuses[vars["taskID"]]
	if !ok {
		http.Error(w, fmt.Sprint("タスクIDが不正です:", vars["taskID"]), http.StatusNotFound)
		return
	}

	if task.result == nil {
		http.Error(w, "実行中タスクは削除できません", http.StatusInternalServerError)
		return
	}

	delete(server.taskStatuses, vars["taskID"])
}

func (server *taskRunnerServer) NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/start", server.handleTaskStart).Methods("POST")
	r.HandleFunc("/cancel/{taskID}", server.handleCancel).Methods("POST")
	r.HandleFunc("/status/{taskID}", server.handleTaskStatus).Methods("GET")
	r.HandleFunc("/delete/{taskID}", server.handleDelete).Methods("GET")
	return r
}

func (server *taskRunnerServer) Run() {
	server.resultDone = make(chan *taskResult)
	server.taskStatuses = make(map[string]*taskStatus)

	for {
		select {
		case result := <-server.resultDone:
			log.Printf("Complete Task. TaskID:%v Success:%v ReturnValues:%v\n", result.ID, result.Success, result.ResultValues)
			server.taskStatusesLock.Lock()
			_, ok := server.taskStatuses[result.ID]
			if ok {
				server.taskStatuses[result.ID].result = result
			}
			server.taskStatusesLock.Unlock()
		}
	}
}
