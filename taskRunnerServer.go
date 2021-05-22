package gojobcoordinatortest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TaskRunnerServer タスク実行管理サーバー
type TaskRunnerServer struct {
	resultDone        chan *TaskResult
	taskStatuses      sync.Map
	taskFactories     sync.Map
	activeTaskNumLock sync.Mutex
	activeTaskNum     uint
	taskNumMax        uint
}

// NewTaskRunnerServer 指定した最大同時タスク実行数のTaskRunnerServerを作成する
func NewTaskRunnerServer(taskNumMax uint) *TaskRunnerServer {
	return &TaskRunnerServer{taskNumMax: taskNumMax, resultDone: make(chan *TaskResult)}
}

// AddFactory タスクファクトリーの登録
func (server *TaskRunnerServer) AddFactory(procName string, f TaskFactoryFunc) error {
	_, exist := server.taskFactories.Load(procName)
	if exist {
		return fmt.Errorf("%sに対応するファクトリはすでに登録されています", procName)
	}

	if f == nil {
		return fmt.Errorf("%sに登録されるファクトリがnilです", procName)
	}

	server.taskFactories.Store(procName, f)
	return nil
}

// NewHTTPHandler HTTPHandlerの作成
func (server *TaskRunnerServer) NewHTTPHandler() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/start", server.handleTaskStart).Methods("POST")
	r.HandleFunc("/cancel/{taskID}", server.handleCancel).Methods("POST")
	r.HandleFunc("/status/{taskID}", server.handleTaskStatus).Methods("GET")
	r.HandleFunc("/delete/{taskID}", server.handleDelete).Methods("GET")
	r.HandleFunc("/alive", server.handleAlive).Methods("GET")
	return r
}

// Run サーバー起動
func (server *TaskRunnerServer) Run() {
	server.RunWithContext(context.Background())
}

// RunWithContext キャンセルコンテキスト指定ありのサーバー起動
func (server *TaskRunnerServer) RunWithContext(ctx context.Context) {
	for {
		select {
		case result := <-server.resultDone:
			log.Printf("Complete Task. TaskID:%v Success:%v ReturnValues:%v\n", result.ID, result.Success, result.ResultValues)
			task, ok := server.getTaskStatus(result.ID)
			if ok {
				task.result = result
			} else {
				log.Print("失敗")
			}

			server.activeTaskNumLock.Lock()
			server.activeTaskNum--
			server.activeTaskNumLock.Unlock()
		case <-ctx.Done():
			log.Print("サーバーを停止します")
			return
		}
	}
}

type taskStatus struct {
	result  *TaskResult
	cancel  context.CancelFunc
	reqData TaskStartRequest
}

func (server *TaskRunnerServer) handleTaskStart(w http.ResponseWriter, r *http.Request) {

	server.activeTaskNumLock.Lock()
	defer server.activeTaskNumLock.Unlock()

	if server.activeTaskNum >= server.taskNumMax {
		http.Error(w, "タスク実行数制限にひっかかりました", http.StatusInternalServerError)
		return
	}

	var requestData TaskStartRequest
	ok := ReadJSONFromRequest(w, r, &requestData)
	if !ok {
		return
	}

	// タスク作成
	task, err := server.newTask(&requestData)
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
	result := TaskStartResponse{ID: taskID}
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, fmt.Sprint("タスク作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		return
	}

	// タスク状態管理情報の作成
	ctx, cancel := context.WithCancel(context.Background())
	server.taskStatuses.Store(taskID, &taskStatus{reqData: requestData, result: nil, cancel: cancel})

	// タスク実行数を加算
	server.activeTaskNum++

	// タスク実行
	// タスクが完了すればserver.resultDoneチャネルに結果が送られる
	// タスクをキャンセルする場合はserver.taskStatusesに保存しているキャンセル関数を呼ぶ
	go task.Run(ctx, taskID, server.resultDone)
	log.Printf("Start Task.　TaskID:%v ProcName:%v Params:%v\n", taskID, requestData.ProcName, requestData.Params)
}

func (server *TaskRunnerServer) handleCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	task, ok := server.getTaskStatus(vars["taskID"])
	if !ok {
		http.Error(w, fmt.Sprint("タスク取得に失敗:", vars["taskID"]), http.StatusNotFound)
		return
	}

	task.cancel()
}

func (server *TaskRunnerServer) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	task, ok := server.getTaskStatus(vars["taskID"])
	if !ok {
		http.Error(w, fmt.Sprint("タスク取得に失敗:", vars["taskID"]), http.StatusNotFound)
		return
	}

	var response TaskStatusResponse
	response.TaskStartRequest = task.reqData
	if task.result != nil {
		if task.result.Success {
			response.Status = StatusSuccess
		} else {
			response.Status = StatusFailure
		}
		response.ResultValues = task.result.ResultValues
	} else {
		response.Status = StatusBusy
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
	}
}

func (server *TaskRunnerServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	task, ok := server.getTaskStatus(vars["taskID"])
	if !ok {
		http.Error(w, fmt.Sprint("タスク取得に失敗:", vars["taskID"]), http.StatusNotFound)
		return
	}

	if task.result == nil {
		http.Error(w, "実行中タスクは削除できません", http.StatusInternalServerError)
		return
	}

	server.taskStatuses.Delete(vars["taskID"])
}

func (server *TaskRunnerServer) handleAlive(w http.ResponseWriter, r *http.Request) {
	return
}

func (server *TaskRunnerServer) newTask(req *TaskStartRequest) (Task, error) {
	factory, ok := server.taskFactories.Load(req.ProcName)
	if !ok {
		return nil, fmt.Errorf("%sに対応するファクトリが存在しません", req.ProcName)
	}

	task, err := factory.(TaskFactoryFunc)(req)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (server *TaskRunnerServer) getTaskStatus(taskID string) (*taskStatus, bool) {
	value, ok := server.taskStatuses.Load(taskID)
	if !ok {
		return nil, false
	}

	task, ok := value.(*taskStatus)
	if !ok {
		return nil, false
	}

	return task, true
}
