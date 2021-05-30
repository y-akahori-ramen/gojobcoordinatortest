package gojobcoordinatortest

import (
	"io"

	"github.com/nsqio/go-nsq"
)

const TaskTopicName string = "Task"
const JobTopicName string = "Job"

// NSQWriter io.WriterをラップしNSQへのログ出力を加える
type NSQWriter struct {
	pub       *nsq.Producer
	topicName string
	writer    io.Writer
}

func NewNSQWriter(nsqdUri, topicName string, writer io.Writer) (*NSQWriter, error) {
	pub, err := nsq.NewProducer(nsqdUri, nsq.NewConfig())
	if err != nil {
		return nil, err
	}
	err = pub.Ping()
	if err != nil {
		return nil, err
	}

	ins := &NSQWriter{pub: pub, topicName: topicName, writer: writer}
	return ins, nil
}

func (w *NSQWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)

	nsqerr := w.pub.Publish(w.topicName, p)
	if nsqerr != nil {
		return 0, nsqerr
	}

	return n, err
}
