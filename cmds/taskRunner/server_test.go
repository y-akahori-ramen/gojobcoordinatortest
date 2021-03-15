package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

func TestStartEchoTask(t *testing.T) {
	server := newTaskRunnerServer(2)
	router := server.newRouter()
	go server.run()

	params := map[string]interface{}{
		"Value": "EchoValue",
	}
	reqestData := gojobcoordinatortest.TaskStartRequest{ProcName: ProcNameEcho, Params: &params}

	req, err := gojobcoordinatortest.NewJSONRequest(http.MethodPost, "/start", reqestData)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("%d != %d, want %d", response.Code, http.StatusOK, http.StatusOK)
	}
}

func TestStartWaitTask(t *testing.T) {
	server := newTaskRunnerServer(2)
	router := server.newRouter()
	go server.run()

	params := map[string]interface{}{
		"Sec": 2.1,
	}
	reqestData := gojobcoordinatortest.TaskStartRequest{ProcName: ProcNameWait, Params: &params}

	req, err := gojobcoordinatortest.NewJSONRequest(http.MethodPost, "/start", reqestData)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		body, err := ioutil.ReadAll(response.Result().Body)
		if err == nil {
			t.Log(string(body))
		}
		t.Fatalf("%d != %d, want %d", response.Code, http.StatusOK, http.StatusOK)
	}

	var result gojobcoordinatortest.TaskStartResponse
	err = gojobcoordinatortest.ReadJSONFromResponse(response.Result(), &result)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go waitTaskComplete(t, router, result.ID, &wg)
	wg.Wait()
}

func TestCancelTask(t *testing.T) {
	server := newTaskRunnerServer(2)
	router := server.newRouter()
	go server.run()

	params := map[string]interface{}{
		"Sec": 2.1,
	}
	reqestData := gojobcoordinatortest.TaskStartRequest{ProcName: ProcNameWait, Params: &params}

	req, err := gojobcoordinatortest.NewJSONRequest(http.MethodPost, "/start", reqestData)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		body, err := ioutil.ReadAll(response.Result().Body)
		if err == nil {
			t.Log(string(body))
		}
		t.Fatalf("%d != %d, want %d", response.Code, http.StatusOK, http.StatusOK)
	}

	var result gojobcoordinatortest.TaskStartResponse
	err = gojobcoordinatortest.ReadJSONFromResponse(response.Result(), &result)
	if err != nil {
		t.Fatal(err)
	}

	req, err = http.NewRequest(http.MethodPost, fmt.Sprint("/cancel/", result.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		body, err := ioutil.ReadAll(response.Result().Body)
		if err == nil {
			t.Log(string(body))
		}
		t.Fatalf("%d != %d, want %d", response.Code, http.StatusOK, http.StatusOK)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go waitTaskComplete(t, router, result.ID, &wg)
	wg.Wait()
}

func TestDeleteTask(t *testing.T) {
	server := newTaskRunnerServer(2)
	router := server.newRouter()
	go server.run()

	params := map[string]interface{}{
		"Sec": 3,
	}
	reqestData := gojobcoordinatortest.TaskStartRequest{ProcName: ProcNameWait, Params: &params}

	req, err := gojobcoordinatortest.NewJSONRequest(http.MethodPost, "/start", reqestData)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("%d != %d, want %d", response.Code, http.StatusOK, http.StatusOK)
	}

	var result gojobcoordinatortest.TaskStartResponse
	err = gojobcoordinatortest.ReadJSONFromResponse(response.Result(), &result)
	if err != nil {
		t.Fatal(err)
	}

	// タスク実行中のキャンセルは失敗する
	req, err = http.NewRequest(http.MethodGet, fmt.Sprint("/delete/", result.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusInternalServerError {
		t.Fatal()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go waitTaskComplete(t, router, result.ID, &wg)
	wg.Wait()

	// タスク完了後の削除は成功する
	response = httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatal()
	}

	// タスク削除後はタスクのステータス取得に失敗する
	req, err = http.NewRequest(http.MethodGet, fmt.Sprint("/status/", result.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusNotFound {
		t.Fatal()
	}

	// タスク削除後はタスクキャンセルに失敗する
	req, err = http.NewRequest(http.MethodPost, fmt.Sprint("/cancel/", result.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusNotFound {
		t.Fatal()
	}
}

func waitTaskComplete(t *testing.T, handler http.Handler, taskID string, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprint("/status/", taskID), nil)
	if err != nil {
		t.Fatal(err)
	}

	for range ticker.C {

		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)

		if response.Code != http.StatusOK {
			body, err := ioutil.ReadAll(response.Result().Body)
			if err == nil {
				t.Log(string(body))
			}
			t.Fatalf("%d != %d, want %d", response.Code, http.StatusOK, http.StatusOK)
		}

		var result gojobcoordinatortest.TaskStatusResponse
		err = gojobcoordinatortest.ReadJSONFromResponse(response.Result(), &result)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != gojobcoordinatortest.StatusBusy {
			if result.ResultValues != nil {
				log.Print(result.ResultValues)
			}
			return
		}
	}
}
