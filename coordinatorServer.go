package gojobcoordinatortest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// CoordinatorServer Coordinatorにサーバー機能を持たせたもの
type CoordinatorServer struct {
	cod *Coordinator
}

// NewCoordinatorServer CoordinatorServerの作成
func NewCoordinatorServer(cod *Coordinator) *CoordinatorServer {
	return &CoordinatorServer{cod: cod}
}

// NewHTTPHandler HTTPHandlerの作成
func (codServer *CoordinatorServer) NewHTTPHandler() http.Handler {
	r := mux.NewRouter()

	// ジョブスタート
	r.HandleFunc("/start", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var startReq JobStartRequest
		if !ReadJSONFromRequest(rw, r, &startReq) {
			return
		}

		startResp, err := codServer.cod.Start(startReq)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}

		err = json.NewEncoder(rw).Encode(startResp)
		if err != nil {
			http.Error(rw, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		}

	}).Methods("POST")

	// ジョブキャンセル
	r.HandleFunc("/cancel/{jobID}", func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := codServer.cod.Cancel(vars["jobID"])
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("POST")

	// ジョブステータス取得
	r.HandleFunc("/status/{jobID}", func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		jobStatusResp, err := codServer.cod.GetStatus(vars["jobID"])
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}

		err = json.NewEncoder(rw).Encode(jobStatusResp)
		if err != nil {
			http.Error(rw, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		}

	}).Methods("GET")

	// TaskRunner接続
	r.HandleFunc("/connect", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var connectionReq TaskRunnerConnectionRequest
		if !ReadJSONFromRequest(rw, r, &connectionReq) {
			return
		}

		err := codServer.cod.Connect(connectionReq)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

	}).Methods("POST")

	// TaskRunner接続解除
	r.HandleFunc("/disconnect", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var connectionReq TaskRunnerConnectionRequest
		if !ReadJSONFromRequest(rw, r, &connectionReq) {
			return
		}

		err := codServer.cod.Disconnect(connectionReq)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

	}).Methods("POST")

	// 接続しているRunner取得
	r.HandleFunc("/runners", func(rw http.ResponseWriter, r *http.Request) {
		responseData := codServer.cod.GetRunners()
		err := json.NewEncoder(rw).Encode(responseData)
		if err != nil {
			http.Error(rw, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		}
	}).Methods("GET")

	// ジョブ一覧取得
	r.HandleFunc("/jobs", func(rw http.ResponseWriter, r *http.Request) {
		responseData := codServer.cod.GetJobs()
		err := json.NewEncoder(rw).Encode(responseData)
		if err != nil {
			http.Error(rw, fmt.Sprint("レスポンス作成に失敗しました:", err.Error()), http.StatusInternalServerError)
		}
	}).Methods("GET")

	return r
}

// Run サーバー起動
func (codServer *CoordinatorServer) Run(ctx context.Context) {
	codServer.cod.Run(ctx)
}
