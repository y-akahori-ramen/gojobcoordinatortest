package gojobcoordinatortest

import "encoding/json"

// MapToStruct map型から構造体に変換する
// タスク開始リクエスト・終了レスポンスの値がマップ型で入っているためそれを構造体に変換する際に使用する
func MapToStruct(mapData map[string]interface{}, v interface{}) error {
	jsonStr, err := json.Marshal(mapData)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonStr, v)
}

// ToMap map型へ変換する
// タスク開始リクエスト・終了レスポンスの値がマップ型で指定するため構造体を指定する際に使用する
func StructToMap(v interface{}) (map[string]interface{}, error) {
	var mapData map[string]interface{}
	jsonStr, err := json.Marshal(v)
	if err != nil {
		return mapData, err
	}

	err = json.Unmarshal(jsonStr, &mapData)
	return mapData, err
}
