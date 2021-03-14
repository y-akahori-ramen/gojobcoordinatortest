# gojobcoordinatortest
時間のかかるジョブを複数クライアントで分散実行させるジョブ管理システムをGo実装してみる

## 構成
TaskRunnerサーバーとCoordinatorサーバーで構成される。  
TaskRunnerサーバーはジョブ実行するためのサーバー。 
Coordinatorサーバーは登録されたTaskRunnerサーバーに対して実行を指示するサーバー。  
TaskRunnerサーバー単体でも使用できるが、Coordinatorサーバーに登録することでCoordinator側に管理されるようになる。  
利用者はCoordinatorに実行依頼を投げるとCoordinatorに登録されたTaskRunnerに実行タスクが振り分けられる。  
利用者はTaskRunnerの存在を知ることなくCoordinatorの存在を知っているだけでよい。

## TaskRunnerAPI
タスクを実行するTaskRunnerサーバーのAPI

### /Start
POSTです。
タスク開始。開始するタスク情報を以下のJSONフォーマットで送る。

```json
{
    "procName":"開始処理名",
    "params":
    {
        "HogeParam":1
    }
}
```

paramsは開始する処理によってはnullの場合もあります。

開始に成功すると `200 OK` の応答があり開始したタスクの情報を以下のJSONフォーマットで受け取る。

```json
{
    "id":"TaskID"
}
```

受け取ったIDで実行したタスクに対して操作を行う。

### /cancel/{taskID}
POSTです。
指定したタスクIDのタスクキャンセルを指示します。

### /status/{taskID}
GETです。
指定したタスクIDのタスク状態を以下のフォーマットで取得します。

```json
{
    "procName": "開始処理名",
    "params": {
        "HogeParam": 1
    },
    "status": "StatusSuccess",
    "resultValues":
    {
        "HogeValue": 1
    }
}
```

タスク開始時に送ったデータに加え、タスクの状態とタスクの結果の値を受け取ります。  
タスクの結果の値はタスクによってはnullの場合があります。  

タスクの状態は以下の３つをとります
- StatusSuccess
    - タスクが成功して終了
- StatusFailure
    - タスクが失敗して終了。キャンセルによって中断された場合もこの値をとります。
- StatusBusy
    - 実行中

### /status/{taskID}
POSTです。
指定したタスクIDのデータを削除します。  
実行中タスクを削除しようとした場合はエラーとなり `500 Internal Server Error` を返します。


