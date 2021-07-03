package main

import (
	"net/http"

	"github.com/gorilla/mux"
	logviewer "github.com/y-akahori-ramen/gojobcoordinatortest/logViewer"
)

// Server ログビューアサーバー
type Server struct {
	renderer *Renderer
}

// NewServer ログビューアサーバーのさ作成
// 表示するログデータへのアクセサを指定する
func NewServer(data logviewer.LogData) (*Server, error) {
	renderer, err := NewRenderer(data)
	if err != nil {
		return nil, err
	}

	return &Server{renderer: renderer}, nil
}

// NewHTTPHandler サーバーへのHTTPリクエストのレスポンスハンドラを作成
func (server *Server) NewHTTPHandler() http.Handler {
	r := mux.NewRouter()

	// indexページ
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := server.renderer.RenderIndex(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	// task一覧ページ
	r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		err := server.renderer.RenderList(w, logviewer.Task)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	// job一覧ページ
	r.HandleFunc("/job", func(w http.ResponseWriter, r *http.Request) {
		err := server.renderer.RenderList(w, logviewer.Job)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	// タスクログページ
	r.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["ID"]
		if !ok {
			http.Error(w, "IDが不正です", http.StatusInternalServerError)
			return
		}
		err := server.renderer.RenderLog(w, logviewer.Task, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	// ジョブログページ
	r.HandleFunc("/job/{ID}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["ID"]
		if !ok {
			http.Error(w, "IDが不正です", http.StatusInternalServerError)
			return
		}
		err := server.renderer.RenderLog(w, logviewer.Job, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	return r
}
