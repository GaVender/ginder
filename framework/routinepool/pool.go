package routinepool

import (
	"time"
	"sync"
	"errors"
	"math"
	"sync/atomic"
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
*** 定义协程池
*/
func NewPool(size, expire uint) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("池的容量参数应大于0")
	}

	pool := &Pool{
		capacity:int32(size),
		expire:time.Second * time.Duration(expire),
		free:make(chan sig, math.MaxInt32),
		release:make(chan sig, 1),
	}

	pool.monitorAndClear()
	return pool, nil
}

/*
*** 添加任务
*/
func (p *Pool) Submit(task f) error {
	if len(p.release) > 0 {
		return errors.New("池已经关闭")
	}

	w := p.getWorker()
	w.sendTask(task)
	return nil
}

/*
*** 获取池的容量
*/
func (p *Pool) Capacity() uint {
	return uint(p.capacity)
}

/*
*** 获取正在运行的worker数量
*/
func (p *Pool) RunningAmount() uint {
	return uint(p.running)
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
		} else {
			p.running++
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

		workers = p.workers
		l := len(workers) - 1
		w = workers[l]
		workers[l] = nil
		p.workers = workers[:l]

		p.lock.Unlock()
	} else if w == nil {
		w = &Worker{
			pool:p,
			task:make(chan f),
		}
		w.run()
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

	go func() {
		for range heartBeat.C {
			currentTime := time.Now()
			p.lock.Lock()

			workers := p.workers
			n := 0

			for i,j := range workers {
				if currentTime.Sub(j.recycleTime) <= p.expire {
					break
				} else {
					n = i
					j.stop()
					workers[i] = nil
					p.running--
				}
			}

			if n > 0 {
				n++
				p.workers = workers[n:]
			}

			p.lock.Unlock()
		}
	}()
}