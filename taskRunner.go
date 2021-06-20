package gojobcoordinatortest

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
)

// TaskRunnerConfig タスクランナーの設定項目
// TaskNumMax タスク同時実行最大数
// Handler タスクのログ出力ハンドリング。不要な場合はnilを指定する。
type TaskRunnerConfig struct {
	TaskNumMax uint
	Handler    LogHandler
}

// TaskRunner タスクの実行管理を行う
type TaskRunner struct {
	TaskRunnerConfig
	resultDone        chan *TaskResult
	taskStatuses      sync.Map
	taskFactories     sync.Map
	activeTaskNumLock sync.Mutex
	activeTaskNum     uint
}

// NewTaskRunner TaskRunnerの作成
func NewTaskRunner(config TaskRunnerConfig) *TaskRunner {
	return &TaskRunner{TaskRunnerConfig: config, resultDone: make(chan *TaskResult)}
}

// AddFactory タスクファクトリーの登録
func (runner *TaskRunner) AddFactory(procName string, f TaskFactoryFunc) error {
	_, exist := runner.taskFactories.Load(procName)
	if exist {
		return fmt.Errorf("%sに対応するファクトリはすでに登録されています", procName)
	}

	if f == nil {
		return fmt.Errorf("%sに登録されるファクトリがnilです", procName)
	}
	runner.taskFactories.Store(procName, f)
	return nil
}

// Run タスクランナー起動
func (runner *TaskRunner) Run(ctx context.Context) {
	for {
		select {
		case result := <-runner.resultDone:
			runner.newTaskLogger(result.ID).Printf("Complete Task. Success:%v ReturnValues:%v\n", result.Success, result.ResultValues)
			task, err := runner.getTaskStatus(result.ID)
			if err != nil {
				log.Print(err.Error())
			} else {
				task.result = result
			}

			runner.activeTaskNumLock.Lock()
			runner.activeTaskNum--
			runner.activeTaskNumLock.Unlock()
		case <-ctx.Done():
			log.Print("TaskRunnerを停止します")
			return
		default:
		}
	}
}

// Start タスクを開始する
func (runner *TaskRunner) Start(req TaskStartRequest) (TaskStartResponse, error) {

	runner.activeTaskNumLock.Lock()
	defer runner.activeTaskNumLock.Unlock()

	if runner.activeTaskNum >= runner.TaskNumMax {
		return TaskStartResponse{}, fmt.Errorf("タスク実行数が上限に達しています Max:%d", runner.TaskNumMax)
	}

	// タスク作成
	task, err := runner.newTask(&req)
	if err != nil {
		return TaskStartResponse{}, err
	}

	// タスクID割り振り
	id, err := uuid.NewRandom()
	if err != nil {
		return TaskStartResponse{}, fmt.Errorf("ID生成に失敗しました:%s", err.Error())
	}
	taskID := id.String()

	// タスク状態管理情報の作成
	ctx, cancel := context.WithCancel(context.Background())
	runner.taskStatuses.Store(taskID, &taskStatus{reqData: req, result: nil, cancel: cancel})

	// タスク実行数を加算
	runner.activeTaskNum++

	// タスク実行
	// タスクが完了すればresultDoneチャネルに結果が送られる
	// タスクをキャンセルする場合はtaskStatusesに保存しているキャンセル関数を呼ぶ
	taskLogger := runner.newTaskLogger(taskID)
	taskLogger.Printf("Start Task. ProcName:%v Params:%v\n", req.ProcName, req.Params)
	go task.Run(ctx, taskID, taskLogger, runner.resultDone)

	return TaskStartResponse{ID: taskID}, nil
}

// CancelReq 指定したタスクにキャンセルリクエストを行う
func (runner *TaskRunner) CancelReq(taskID string) error {
	task, err := runner.getTaskStatus(taskID)
	if err != nil {
		return err
	}

	task.cancel()

	return nil
}

// Delete 指定したタスクを削除する
// 実行中のタスクを削除しようとした場合は失敗する
func (runner *TaskRunner) Delete(taskID string) error {
	task, err := runner.getTaskStatus(taskID)
	if err != nil {
		return err
	}

	if task.result == nil {
		return fmt.Errorf("実行中タスクは削除できません:%s", taskID)
	}

	runner.taskStatuses.Delete(taskID)
	return nil
}

// GetTaskIDs 管理対象のタスクID一覧を取得する
func (runner *TaskRunner) GetTaskIDs() []string {
	var tasks []string
	addTask := func(task, _ interface{}) bool {
		tasks = append(tasks, task.(string))
		return true
	}
	runner.taskStatuses.Range(addTask)
	return tasks
}

// GetTaskStatusResponse 指定したタスクの状態取得
func (runner *TaskRunner) GetTaskStatusResponse(taskID string) (TaskStatusResponse, error) {
	var response TaskStatusResponse

	status, err := runner.getTaskStatus(taskID)
	if err != nil {
		return response, err
	}

	response.TaskStartRequest = status.reqData
	if status.result != nil {
		if status.result.Success {
			response.Status = StatusSuccess
		} else {
			response.Status = StatusFailure
		}
		response.ResultValues = status.result.ResultValues
	} else {
		response.Status = StatusBusy
	}

	return response, nil
}

type taskStatus struct {
	result  *TaskResult
	cancel  context.CancelFunc
	reqData TaskStartRequest
}

// getTaskStatus 指定したタスク状態を取得する
func (runner *TaskRunner) getTaskStatus(taskID string) (*taskStatus, error) {
	value, ok := runner.taskStatuses.Load(taskID)
	if !ok {
		return nil, fmt.Errorf("タスク取得に失敗:%s", taskID)
	}

	task, ok := value.(*taskStatus)
	if !ok {
		return nil, fmt.Errorf("タスク取得に失敗:%s", taskID)
	}

	return task, nil
}

func (runner *TaskRunner) newTask(req *TaskStartRequest) (Task, error) {
	factory, ok := runner.taskFactories.Load(req.ProcName)
	if !ok {
		return nil, fmt.Errorf("%sに対応するファクトリが存在しません", req.ProcName)
	}

	task, err := factory.(TaskFactoryFunc)(req)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (runner *TaskRunner) newTaskLogger(taskID string) *log.Logger {
	writer := &WriterWithHandler{Writer: log.Default().Writer(), Handler: runner.Handler, Id: taskID}
	return log.New(writer, fmt.Sprintf("[%s]", taskID), log.Default().Flags())
}
