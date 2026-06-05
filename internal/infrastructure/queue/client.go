package queue

import (
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

// Client wraps asynq.Client for enqueueing tasks.
type Client struct {
	asynq *asynq.Client
	log   *logrus.Logger
}

// NewClient creates a new queue client.
func NewClient(redisOpt asynq.RedisClientOpt, log *logrus.Logger) *Client {
	return &Client{
		asynq: asynq.NewClient(redisOpt),
		log:   log,
	}
}

// Enqueue sends a task to the queue and returns the task info.
func (c *Client) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	info, err := c.asynq.Enqueue(task, opts...)
	if err != nil {
		return nil, err
	}
	c.log.WithFields(logrus.Fields{
		"task_id":   info.ID,
		"task_type": info.Type,
		"queue":     info.Queue,
	}).Info("task enqueued")
	return info, nil
}

// Close shuts down the client.
func (c *Client) Close() error {
	return c.asynq.Close()
}
