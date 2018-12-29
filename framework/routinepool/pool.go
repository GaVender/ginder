package routinepool

import (
	"time"
	"sync"
	"errors"
	"math"
)

type f func() error
type sig struct{}

type Pool struct {
	// pool 的容量，即可生成的 worker 最大数量
	capacity uint32

	// 正在运行的 worker 数量
	running uint32

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

func NewPool(size, expire uint) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("池的大小设置有误")
	}

	pool := &Pool{
		capacity:uint32(size),
		expire:time.Second * time.Duration(expire),
		free:make(chan sig, math.MaxInt32),
		release:make(chan sig, 1),
	}

	return pool, nil
}

func (p *Pool) Submit(task f) error {
	if len(p.release) > 0 {
		return errors.New("池已经关闭")
	}

	w := p.getWorker()
	w.sendTask(task)
	return nil
}

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
	}

	return w
}

func (p *Pool) putWorker(w *Worker) {
	w.recycleTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.lock.Unlock()
	p.free <- sig{}
}