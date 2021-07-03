package main

import (
	"errors"
	"html/template"
	"io"
	"path"
	"strings"

	logviewer "github.com/y-akahori-ramen/gojobcoordinatortest/logViewer"
)

// Renderer 保存されたログのHTMLレンダラ
type Renderer struct {
	data          logviewer.LogData
	indexTemplate *template.Template
	listTemplate  *template.Template
	logTemplate   *template.Template
}

// NewRenderer ログのHTMLレンダラ作成
// dataにはログ情報へのアクセサを使用する
func NewRenderer(data logviewer.LogData) (*Renderer, error) {
	funcs := template.FuncMap{
		"log": func(text string) template.HTML {
			// ログ表示時に指定する関数
			// [69fe5b5d-1469-4b34-86e2-fdc7d589ba93]2021/06/26 23:06:45 Start Task.
			// というように最初にIDが入っているがログページのタイトルからIDがわかるため取り除く
			idEndIndex := strings.Index(text, "]")
			id := text[:idEndIndex+1]
			removedIdStr := strings.ReplaceAll(text, id, "")
			// 改行コードを<br>に変換
			return template.HTML(strings.ReplaceAll(template.HTMLEscapeString(removedIdStr), "\n", "<br>"))
		},
	}
	logTemplate, err := template.New("log.html").Funcs(funcs).ParseFiles("template/log.html")
	if err != nil {
		return nil, err
	}
	listTemplate, err := template.New("list.html").Funcs(funcs).ParseFiles("template/list.html")
	if err != nil {
		return nil, err
	}
	indexTemplate, err := template.New("index.html").Funcs(funcs).ParseFiles("template/index.html")
	if err != nil {
		return nil, err
	}

	return &Renderer{data: data, logTemplate: logTemplate, listTemplate: listTemplate, indexTemplate: indexTemplate}, nil
}

// RenderIndex インデックスページのレンダリング
func (r *Renderer) RenderIndex(wr io.Writer) error {
	data := struct {
		TaskListURI string
		JobListURI  string
	}{
		TaskListURI: "task",
		JobListURI:  "job",
	}
	return r.indexTemplate.Execute(wr, data)
}

// RenderList ログ一覧ページのレンダリング
func (r *Renderer) RenderList(wr io.Writer, dataType logviewer.DataType) error {
	var idList []string
	var err error
	var logUrlBase, title string
	switch dataType {
	case logviewer.Task:
		title = "TaskList"
		logUrlBase = "task"
		idList, err = r.data.GetTaskList()
		if err != nil {
			return err
		}
	case logviewer.Job:
		title = "JobList"
		logUrlBase = "job"
		idList, err = r.data.GetJobList()
		if err != nil {
			return err
		}
	default:
		return errors.New("データタイプが不正です")
	}

	type logListData struct {
		ID  string
		URI string
	}

	var logListDatas []logListData
	for _, id := range idList {
		logListDatas = append(logListDatas, logListData{ID: id, URI: path.Join(logUrlBase, id)})
	}

	data := struct {
		Title   string
		HomeURI string
		Logs    []logListData
	}{
		Title:   title,
		HomeURI: ".",
		Logs:    logListDatas,
	}
	return r.listTemplate.Execute(wr, data)
}

// RenderLog ログデータのレンダリング
func (r *Renderer) RenderLog(wr io.Writer, dataType logviewer.DataType, id string) error {
	log, err := r.data.GetTaskLog(id)
	if err != nil {
		return err
	}

	var title, listURI, listName string
	switch dataType {
	case logviewer.Task:
		title = "TaskLog:" + id
		listURI = "../task"
		listName = "TaskList"
	case logviewer.Job:
		title = "JobLog:" + id
		listURI = "../job"
		listName = "JobList"
	default:
		return errors.New("データタイプが不正です")
	}

	data := struct {
		Title    string
		Log      string
		HomeURI  string
		ListURI  string
		LogID    string
		ListName string
	}{
		Title:    title,
		Log:      log,
		HomeURI:  "..",
		ListURI:  listURI,
		LogID:    id,
		ListName: listName,
	}

	return r.logTemplate.Execute(wr, data)
}
