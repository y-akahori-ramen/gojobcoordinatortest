package gojobcoordinatortest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// TaskRunnerServer TaskRunnerにサーバー機能を持たせたもの
type TaskRunnerServer struct {
	runner *TaskRunner
}

// NewTaskRunnerServer 指定したTaskRunnerのサーバーを作成する
func NewTaskRunnerServer(runner *TaskRunner) *TaskRunnerServer {
	return &TaskRunnerServer{runner: runner}
}

// NewHTTPHandler HTTPHandlerの作成
func (server *TaskRunnerServer) NewHTTPHandler() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/start", server.handleTaskStart).Methods("POST")
	r.HandleFunc("/cancel/{taskID}", server.handleCancel).Methods("POST")
	r.HandleFunc("/status/{taskID}", server.handleTaskStatus).Methods("GET")
	r.HandleFunc("/delete/{taskID}", server.handleDelete).Methods("POST")
	r.HandleFunc("/alive", server.handleAlive).Methods("GET")
	r.HandleFunc("/tasks", server.handleTasks).Methods("GET")
	return r
}

// Run サーバー起動
func (server *TaskRunnerServer) Run(ctx context.Context) {
	server.runner.Run(ctx)
}

func (server *TaskRunnerServer) handleTaskStart(w http.ResponseWriter, r *http.Request) {

	var requestData TaskStartRequest
	ok := ReadJSONFromRequest(w, r, &requestData)
	if !ok {
		http.Error(w, "JSONパースに失敗しました", http.StatusBadRequest)
		return
	}

	response, err := server.runner.Start(requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// タスクIDが確定するのでここでレスポンス作成　※タスク開始後にレスポンス作成失敗すると開始したタスクを止められないため
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, fmt.Sprint("JSONエンコードに失敗しました:", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (server *TaskRunnerServer) handleCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := server.runner.CancelReq(vars["taskID"]); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
}

func (server *TaskRunnerServer) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	statusResponse, err := server.runner.GetTaskStatusResponse(vars["taskID"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	err = json.NewEncoder(w).Encode(statusResponse)
	if err != nil {
		http.Error(w, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
	}
}

func (server *TaskRunnerServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := server.runner.Delete(vars["taskID"])

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (server *TaskRunnerServer) handleAlive(w http.ResponseWriter, r *http.Request) {
	return
}

func (server *TaskRunnerServer) handleTasks(w http.ResponseWriter, r *http.Request) {

	taskIDs := server.runner.GetTaskIDs()

	response := TaskListResponse{Tasks: taskIDs}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
	}

	return
}
