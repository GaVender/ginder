package routinepool

import (
	"time"
	"sync"
	"sync/atomic"
	"fmt"
)

type f func() error
type sig struct{}

/*
*** 池的结构
*/
type Pool struct {
	// pool 的容量，即可生成的 worker 最大数量
	capacity int32

	// 正在运行的 worker 数量
	running int32

	// 设置每个 worker 的过期时间（秒）
	expire time.Duration

	// 标识 pool 有空闲的 workers 可以用来工作
	free chan sig

	// 存储空闲的 worker
	workers []*Worker

	// 关闭该 pool 支持通知所有 worker 退出运行，以防 goroutine 泄露
	release chan sig

	// 支持 pool 的同步操作
	lock sync.Mutex

	// 确保 pool 关闭操作只会执行一次
	once sync.Once
}

/*
*** 新建池
*/
func NewPool(size, expire uint) (*Pool, error) {
	if size <= 0 {
		return nil, ErrPoolCapacity
	}

	if expire <= 0 {
		return nil, ErrPoolExpire
	}

	pool := &Pool{
		capacity:int32(size),
		expire:time.Second * time.Duration(expire),
		free:make(chan sig, size),
		release:make(chan sig, 1),
	}

	go pool.monitorAndClear()
	return pool, nil
}

/*
*** 打开池
*/
func (p *Pool) Open() {
	if len(p.release) > 0 {
		<- p.release
	}
}

/*
*** 关闭池
*/
func (p *Pool) Close() {
	p.once.Do(func() {
		p.release <- sig{}
		p.lock.Lock()
		workers := p.workers

		for i, j := range workers {
			j.stop()
			workers[i] = nil
		}

		p.workers = nil
		p.lock.Unlock()
	})
}

/*
*** 添加任务
*/
func (p *Pool) Submit(task f) error {
	if len(p.release) > 0 {
		return ErrPoolClosed
	}

	w := p.getWorker()
	w.sendTask(task)
	return nil
}

/*
*** 获取池的容量
*/
func (p *Pool) Capacity() uint {
	return uint(atomic.LoadInt32(&p.capacity))
}

/*
*** 获取正在运行的worker数量
*/
func (p *Pool) RunningAmount() uint {
	return uint(atomic.LoadInt32(&p.running))
}

/*
*** 获取正在运行的worker数量
*/
func (p *Pool) FreeAmount() uint {
	return uint(atomic.AddInt32(&p.capacity, -atomic.LoadInt32(&p.running)))
}

/*
*** 调整池的容量
*/
func (p *Pool) Resize(size uint) {
	if size < p.Capacity() {
		diff := p.Capacity() - size

		for i := 0; uint(i) < diff; i++ {
			p.getWorker().stop()
		}
	} else if size == p.Capacity() {
		return
	}

	atomic.StoreInt32(&p.capacity, int32(size))
}

/*
*** 增加正在运行的worker数量
*/
func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

/*
*** 减少正在运行的worker数量
*/
func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}

/*
*** 获取worker
*** 1、无空闲的worker，若worker数量等于池的容量，则等待其他worker跑完；若小于，则新开一个
*** 2、有空闲的worker，拿出来使用
 */
func (p *Pool) getWorker() *Worker {
	var w *Worker
	waiting := false
	p.lock.Lock()

	workers := p.workers
	n := len(workers) - 1

	if n < 0 {
		if p.running >= p.capacity {
			waiting = true
		}
	} else {
		<- p.free
		w = p.workers[n]
		workers[n] = nil
		p.workers = workers[:n]
	}

	p.lock.Unlock()

	if waiting {
		<- p.free
		p.lock.Lock()

		l := len(p.workers) - 1
		w = p.workers[l]
		workers = p.workers
		workers[l] = nil
		p.workers = workers[:l]

		p.lock.Unlock()
	} else if w == nil {
		w = &Worker{
			pool:p,
			task:make(chan f),
		}
		w.run()
		p.incRunning()
	}

	return w
}

/*
*** 回收worker
 */
func (p *Pool) putWorker(w *Worker) {
	w.recycleTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.lock.Unlock()
	p.free <- sig{}
}

/*
*** 监控并定期清理worker
 */
func (p *Pool) monitorAndClear() {
	heartBeat := time.NewTicker(p.expire)
	defer heartBeat.Stop()

	for range heartBeat.C {
		currentTime := time.Now()
		p.lock.Lock()

		workers := p.workers

		if len(workers) == 0 && p.RunningAmount() == 0 && len(p.release) > 0 {
			p.lock.Unlock()
			return
		} else {
			var _ = fmt.Println
			//fmt.Println("worker 个数：", len(workers))
		}

		n := 0

		for i, j := range workers {
			if j != nil {
				if currentTime.Sub(j.recycleTime) <= p.expire {
					break
				} else {
					n = i
					j.stop()
					workers[i] = nil
				}
			}
		}

		if n > 0 {
			p.workers = workers[n + 1:]
		}

		p.lock.Unlock()
	}
}