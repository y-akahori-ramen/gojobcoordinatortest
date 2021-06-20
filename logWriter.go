package gojobcoordinatortest

import "io"

// WriterWithHandler wrtierをラップしハンドラーへwriterに書き込む内容を渡す。このio.Writerをログ出力先として使用する
type WriterWithHandler struct {
	Writer  io.Writer
	Handler LogHandler
	Id      string
}

func (w *WriterWithHandler) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)

	if w.Handler != nil {
		w.Handler.HandleLog(w.Id, p)
	}

	return n, err
}
