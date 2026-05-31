package worker

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type Task func()

type Pool struct {
	tasks chan Task
	wg    sync.WaitGroup
	log   *logrus.Logger
}

func NewPool(maxWorkers int, log *logrus.Logger) *Pool {
	return &Pool{
		tasks: make(chan Task, maxWorkers*10),
		log:   log,
	}
}

func (p *Pool) Start(maxWorkers int) {
	for i := 0; i < maxWorkers; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			for task := range p.tasks {
				task()
			}
		}(i)
	}
	p.log.WithField("max_workers", maxWorkers).Info("worker pool started")
}

// Submit adds a task to the pool. Blocks if the task buffer is full.
func (p *Pool) Submit(task Task) {
	p.tasks <- task
}

// Shutdown waits for all running goroutines to finish.
func (p *Pool) Shutdown() {
	close(p.tasks)
	p.wg.Wait()
	p.log.Info("worker pool shut down")
}
