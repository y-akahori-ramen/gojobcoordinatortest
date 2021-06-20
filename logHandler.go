package gojobcoordinatortest

// LogHandler ログ出力のハンドリングインターフェース
// タスク・ジョブのログ出力時に呼び出されます。スレッドセーフである必要があります。
type LogHandler interface {
	// Write ログ出力を行ったタスク・ジョブIDと出力した内容を受け取る
	HandleLog(id string, p []byte)
}
