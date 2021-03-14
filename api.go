package gojobcoordinatortest

// API用のJSONフォーマット

// TaskStartRequest TaskRunnerにタスク開始リクエストを行う時のリクエストデータ
type TaskStartRequest struct {
	ProcName string                  `json:"procName"`
	Params   *map[string]interface{} `json:"params"`
}

// TaskStartResponse TaskRunnerにタスク開始APIを叩いた時のレスポンス
type TaskStartResponse struct {
	ID string `json:"id"`
}

// TaskStatusResponse TaskRunnerにタスクの状態確認APIを叩いた時のレスポンス
type TaskStatusResponse struct {
	TaskStartRequest
	Status       string                  `json:"status"`
	ResultValues *map[string]interface{} `json:"resultValues"`
}

const (
	// StatusSuccess Taskが成功して終了している時にTaskStatusResponseのStatusで返される値
	StatusSuccess string = "StatusSuccess"
	// StatusFailure Taskが失敗して終了いる時にTaskStatusResponseのStatusで返される値
	StatusFailure string = "StatusFailure"
	// StatusBusy Taskが実行中な時にTaskStatusResponseのStatusで返される値
	StatusBusy string = "StatusBusy"
)

// TaskRunnerConnectionRequest コーディネーターサーバーにTaskRunnerを接続・解除する際のリクエスト
type TaskRunnerConnectionRequest struct {
	Address string `json:"address"`
}

// JobStartRequest コーディネーターサーバーに送るジョブ開始リクエスト
type JobStartRequest struct {
	Tasks []TaskStartRequest `json:"tasks"`
}

// JobStartResponse コーディネーターサーバーへジョブ開始リクエストを行った時のレスポンス
type JobStartResponse struct {
	ID string `json:"id"`
}

// JobStatusResponse コーディネーターサーバーへジョブ状態取得を行った時のレスポンス
type JobStatusResponse struct {
	Busy         bool                  `json:"busy"`
	TaskStatuses *[]TaskStatusResponse `json:"taskStatuses"`
}
