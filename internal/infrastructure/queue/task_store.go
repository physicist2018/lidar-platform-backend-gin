package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const taskResultTTL = 1 * time.Hour

// TaskResult holds the outcome of an async task for polling.
type TaskResult struct {
	Status    string `json:"status"` // pending, processing, done, failed
	URL       string `json:"url,omitempty"`
	Error     string `json:"error,omitempty"`
	UpdatedAt int64  `json:"updated_at"`
}

// TaskStore persists task results in Redis for polling.
type TaskStore struct {
	rdb *redis.Client
}

// NewTaskStore creates a new task result store.
func NewTaskStore(rdb *redis.Client) *TaskStore {
	return &TaskStore{rdb: rdb}
}

func taskResultKey(taskID string) string {
	return "task:result:" + taskID
}

// Set writes the task result to Redis.
func (s *TaskStore) Set(ctx context.Context, taskID string, res *TaskResult) error {
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, taskResultKey(taskID), data, taskResultTTL).Err()
}

// Get reads the task result from Redis.
func (s *TaskStore) Get(ctx context.Context, taskID string) (*TaskResult, error) {
	data, err := s.rdb.Get(ctx, taskResultKey(taskID)).Result()
	if err != nil {
		return nil, err
	}
	var res TaskResult
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		return nil, err
	}
	return &res, nil
}
