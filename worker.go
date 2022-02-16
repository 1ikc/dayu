package dayu

import (
	"github.com/google/uuid"
	"log"
	"time"
)

type Worker struct {
	// id workerID
	id string
	// pool 所属协程池
	pool *Pool
	// task 任务通道
	task chan f
	// recycleTime 回收时间
	recycleTime time.Time
}

func newWorker(p *Pool) *Worker {
	return &Worker{
		id:   uuid.NewString(),
		pool: p,
		task: make(chan f, 1),
	}
}

func (w *Worker) run() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				w.pool.decrRunning()
				log.Printf("worker[%s] panic: %v", w.id, err)
			}
		}()

		for f := range w.task {
			if f == nil || len(w.pool.release) > 0 {
				w.pool.decrRunning()
				return
			}

			if err := f(); err != nil {
				log.Printf("worker[%s] error: %v", w.id, err)
			}

			w.pool.putWorker(w)
		}
	}()
}
