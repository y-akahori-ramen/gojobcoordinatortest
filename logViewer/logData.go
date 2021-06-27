package logviewer

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// LogData 保存されたログへのアクセスインターフェイス
type LogData interface {
	// GetTaskLog 指定したタスクIDのログを時間の昇順で取得します
	GetTaskLog(taskID string) ([]string, error)
	// GetJobLog 指定したジョブIDのログを時間の昇順で取得します
	GetJobLog(jobID string) ([]string, error)
	// GetTaskList タスクID一覧を取得します
	GetTaskList() ([]string, error)
	// GetJobList ジョブID一覧を取得します
	GetJobList() ([]string, error)
}

// MongoLogData MongoDBに保存されたログ情報へのアクセサ
type MongoLogData struct {
	client   *mongo.Client
	taskLogs *mongo.Collection
	taskList *mongo.Collection
	jobLogs  *mongo.Collection
	jobList  *mongo.Collection
}

// Close 接続したDBから切断します
func (data *MongoLogData) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := data.client.Disconnect(ctx); err != nil {
		panic(err)
	}
}

// NewMongoLogData LogDataの作成
// dbUri はログが保存されているデータベースを指定する。
// localhostのポート27017でアクセスできるmongodbに保存しているのであれば以下のようになる
// mongodb://localhost:27017
func NewMongoLogData(dbUri string) (*MongoLogData, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// logViewerデータベースに情報を保存しているためuriにアクセス先を加える
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbUri+"/logViewer"))
	if err != nil {
		return &MongoLogData{}, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return &MongoLogData{}, err
	}

	database := client.Database("logViewer")

	return &MongoLogData{
			client:   client,
			taskLogs: database.Collection("task"),
			taskList: database.Collection("taskStart"),
			jobLogs:  database.Collection("job"),
			jobList:  database.Collection("jobStart")},
		nil
}

// GetTaskLog 指定したタスクIDのログを時間の昇順で取得します
func (data *MongoLogData) GetTaskLog(taskID string) ([]string, error) {
	return getLog(data.taskLogs, taskID)
}

// GetJobLog 指定したジョブIDのログを時間の昇順で取得します
func (data *MongoLogData) GetJobLog(jobID string) ([]string, error) {
	return getLog(data.jobLogs, jobID)
}

// GetTaskList タスクID一覧を取得します
func (data *MongoLogData) GetTaskList() ([]string, error) {
	return getList(data.taskList)
}

// GetJobList ジョブID一覧を取得します
func (data *MongoLogData) GetJobList() ([]string, error) {
	return getList(data.jobList)
}

func getLog(collection *mongo.Collection, id string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var retValue []string
	opts := options.Find()
	opts.SetSort(bson.M{"time": 1})
	filterCursor, err := collection.Find(ctx, bson.M{"id": id}, opts)
	if err != nil {
		return retValue, err
	}

	var logs []bson.M
	if err = filterCursor.All(ctx, &logs); err != nil {
		return retValue, err
	}
	for _, log := range logs {
		v, ok := log["log"]
		if !ok {
			return retValue, errors.New("logのデータが存在しません")
		}
		retValue = append(retValue, v.(string))
	}

	return retValue, nil
}

func getList(collection *mongo.Collection) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var retValue []string
	opts := options.Find()
	opts.SetSort(bson.M{"time": 1})
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return retValue, err
	}

	var list []bson.M
	if err = cursor.All(ctx, &list); err != nil {
		return retValue, err
	}
	for _, elem := range list {
		v, ok := elem["id"]
		if !ok {
			return retValue, errors.New("idのデータが存在しません")
		}
		retValue = append(retValue, v.(string))
	}

	return retValue, nil
}
