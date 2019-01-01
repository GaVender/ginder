package routinepool

import (
	"time"
	"sync/atomic"
)

type Worker struct {
	// 拥有该 worker 的 pool
	pool *Pool

	// 要执行的任务
	task chan f

	// 回收时间，亦即该 worker 的最后运行时间
	recycleTime time.Time
}

func (w *Worker) run() {
	go func() {
		for f := range w.task {
			if f == nil {
				atomic.AddInt32(&w.pool.running, -1)
				return
			}

			f()
			w.pool.putWorker(w)
		}
	}()
}

func (w *Worker) stop() {
	w.sendTask(nil)
}

func (w *Worker) sendTask(task f) {
	w.task <- task
}