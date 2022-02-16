package dayu

import (
	"log"
	"sync"
	"time"
)

type PoolAPI interface {
	Submit(f) error
	ReSize(int) error
	Release()
}

type f func(...interface{}) error

type sig struct{}

type Pool struct {
	noCopy

	// capacity 协程池容量
	capacity int
	// running 运行中的协程
	running int
	// expire worker的过期时间
	expire time.Duration
	// release 关闭信号通道 通知worker关闭 避免goroutine泄漏
	release chan sig
	// idleWorker 空闲协程序列 FILO
	idleWorker []*Worker

	mu   sync.Mutex
	once sync.Once
}

func NewPool(options ...Option) (*Pool, error) {
	p := &Pool{
		capacity: DefaultPoolSize,
		expire:   DefaultCleanIntervalTime * time.Second,
		release:  make(chan sig, 1),
	}

	for _, option := range options {
		if err := option(p); err != nil {
			return nil, err
		}
	}

	log.Println("pool init finish")
	go p.purge()

	return p, nil
}

func (p *Pool) Submit(task f) error {
	if len(p.release) > 0 {
		return ErrPoolClosed
	}

	w := p.getWorker()
	w.task <- task

	return nil
}

func (p *Pool) ReSize(size int) error {
	if size < 0 {
		return ErrInvalidPoolSize
	}

	if size == p.capacity {
		return nil
	}

	p.mu.Lock()
	p.capacity = size
	diff := p.running - size
	p.mu.Unlock()

	for ; diff > 0; diff-- {
		p.getWorker().task <- nil
	}

	return nil
}

func (p *Pool) Release() {
	p.mu.Lock()
	p.release <- sig{}
	for i, w := range p.idleWorker {
		w.task <- nil
		p.idleWorker[i] = nil
	}
	p.idleWorker = p.idleWorker[:0]
	p.mu.Unlock()
}

func (p *Pool) purge() {
	heartBeat := time.NewTicker(p.expire)
	for range heartBeat.C {
		log.Println("pool purge running")
		p.mu.Lock()

		currentTime := time.Now()
		idleWorker := p.idleWorker
		if len(idleWorker) == 0 || p.running == 0 || len(p.release) > 0 {
			p.mu.Unlock()
			return
		}

		n := 0
		for i, w := range idleWorker {
			if currentTime.Sub(w.recycleTime) <= p.expire {
				break
			}

			n = i
			w.task <- nil
			idleWorker[i] = nil
		}
		n++

		if n >= len(idleWorker) {
			p.idleWorker = idleWorker[:0]
		} else {
			p.idleWorker = idleWorker[n:]
		}

		p.mu.Unlock()
	}
}

func (p *Pool) getWorker() *Worker {
	var w *Worker
	waiting := false

	p.mu.Lock()
	idleWorker := p.idleWorker
	n := len(idleWorker) - 1
	if n > 0 {
		w = idleWorker[n]
		idleWorker[n] = nil
		idleWorker = idleWorker[:n]
	} else {
		waiting = p.running >= p.capacity
	}
	p.mu.Unlock()

	if waiting {
		for {
			p.mu.Lock()
			idleWorker := p.idleWorker
			n := len(idleWorker) - 1
			if n > 0 {
				w = idleWorker[n]
				idleWorker[n] = nil
				idleWorker = idleWorker[:n]
				p.mu.Unlock()
				break
			}
			p.mu.Unlock()
		}
	} else if w == nil {
		w = newWorker(p)
		w.run()
		p.incrRunning()
	}

	return w
}

func (p *Pool) putWorker(w *Worker) {
	w.recycleTime = time.Now()
	p.mu.Lock()
	p.idleWorker = append(p.idleWorker, w)
	p.mu.Unlock()
}

func (p *Pool) incrRunning() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running++
}

func (p *Pool) decrRunning() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running--
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
