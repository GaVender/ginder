package routinepool

import (
	"time"
)

/*
*** worker的结构
*/
type Worker struct {
	// 拥有该 worker 的 pool
	pool *Pool

	// 要执行的任务
	task chan f

	// 回收时间，亦即该 worker 的最后运行时间
	recycleTime time.Time
}

/*
*** worker运行
*/
func (w *Worker) run() {
	go func() {
		// chan的属性，会使协程阻塞，除非有值进来，nil也可以
		for f := range w.task {
			if f == nil {
				w.pool.decRunning()
				return
			}

			f()
			w.pool.putWorker(w)
		}
	}()
}

/*
*** worker停止
*/
func (w *Worker) stop() {
	w.sendTask(nil)
}

/*
*** worker添加任务
*/
func (w *Worker) sendTask(task f) {
	w.task <- task
}