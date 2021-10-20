package kubernetes

import (
	"github.com/streadway/amqp"
)

type QueueManager struct {
	conn *amqp.Connection
}

func NewQueueManager(url string) (*QueueManager, error) {
	conn, err := amqp.Dial(url)

	if err != nil {
		return nil, err
	}

	return &QueueManager{conn: conn}, nil
}

func (qm *QueueManager) Close() error {
	return qm.conn.Close()
}

func (qm *QueueManager) WithChannel(f func(channel *amqp.Channel) error) error {
	ch, err := qm.conn.Channel()

	if err != nil {
		return err
	}
	defer ch.Close()

	if err = ch.Qos(1, 0, false); err != nil {
		return err
	}

	return f(ch)
}
