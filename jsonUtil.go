package gojobcoordinatortest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// ReadJSONFromRequest HTTPリクエストのBodyをJSONデータと仮定し、そのJSONを読み込みます
// エラーが発生した場合http.ResponseWriterに書き込まれます
func ReadJSONFromRequest(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type header is not application/json", http.StatusUnsupportedMediaType)
		return false
	}

	if r.ContentLength <= 0 {
		http.Error(w, "Body is None", http.StatusBadRequest)
		return false
	}

	err := json.NewDecoder(r.Body).Decode(&dst)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON format error. %v", err.Error()), http.StatusBadRequest)
		return false
	}

	return true
}

func ReadJSONFromResponse(res *http.Response, dst interface{}) error {

	// if _, err := ioutil.ReadAll(res.Body); err != nil {
	// 	return err
	// }

	if err := json.NewDecoder(res.Body).Decode(&dst); err != nil {
		return err
	}

	return nil
	// return json.NewDecoder(response.Body).Decode(&dst)

	// defer res.Body.Close()

	// b, err := ioutil.ReadAll(res.Body)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(b))

	// var response googleResponse
	// if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
	// 	return nil, err
	// }
	// return &response, nil
}

// NewJSONRequest 指定したデータをJSON形式にしたHTTPリクエストを作成します
func NewJSONRequest(method, url string, sendJSONData interface{}) (*http.Request, error) {
	jsonData, err := json.Marshal(sendJSONData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	return req, err
}
