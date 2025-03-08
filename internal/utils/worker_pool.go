package utils

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// WORKERPOOL MANAGES A POOL OF WORKER GOROUTINES
type WorkerPool struct {
	tasks          chan func() error
	wg             sync.WaitGroup
	size           int
	runningTasks   int32
	completedTasks int32
	failedTasks    int32
	isShutdown     bool
	shutdownOnce   sync.Once
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// NEWWORKERPOOL CREATES A NEW WORKER POOL
func NewWorkerPool(size int) *WorkerPool {
	// SET SENSIBLE DEFAULTS
	if size <= 0 {
		size = 5
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		tasks:      make(chan func() error, size*10), // BUFFER 10X THE WORKER COUNT
		size:       size,
		ctx:        ctx,
		cancel:     cancel,
		isShutdown: false,
		mu:         sync.RWMutex{},
		wg:         sync.WaitGroup{},
	}

	// START WORKERS
	for i := 0; i < size; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// WORKER PROCESSES TASKS FROM THE QUEUE
func (p *WorkerPool) worker() {
	if p == nil {
		return
	}

	defer p.wg.Done()

	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				// CHANNEL CLOSED, EXIT
				return
			}

			// SKIP NIL TASKS
			if task == nil {
				atomic.AddInt32(&p.failedTasks, 1)
				continue
			}

			// INCREMENT RUNNING COUNTER
			atomic.AddInt32(&p.runningTasks, 1)

			// EXECUTE TASK WITH PANIC RECOVERY
			func() {
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt32(&p.failedTasks, 1)
						log.Printf("Worker recovered from panic: %v", r)
					}

					// DECREMENT RUNNING COUNTER
					atomic.AddInt32(&p.runningTasks, -1)
				}()

				err := task()

				// UPDATE STATISTICS
				if err != nil {
					atomic.AddInt32(&p.failedTasks, 1)
				} else {
					atomic.AddInt32(&p.completedTasks, 1)
				}
			}()

		case <-p.ctx.Done():
			// CONTEXT CANCELED, EXIT
			return
		}
	}
}

// SUBMIT ADDS A TASK TO THE POOL
func (p *WorkerPool) Submit(task func() error) error {
	if p == nil {
		return fmt.Errorf("worker pool is nil")
	}

	p.mu.RLock()
	if p.isShutdown {
		p.mu.RUnlock()
		return ErrPoolShutdown
	}
	p.mu.RUnlock()

	// USE NON-BLOCKING SEND WITH FALLBACK TO DIRECT EXECUTION
	select {
	case p.tasks <- task:
		return nil
	case <-p.ctx.Done():
		return ErrPoolShutdown
	default:
		// BUFFER FULL - EXECUTE DIRECTLY
		err := task()
		// UPDATE COUNTERS DIRECTLY
		if err != nil {
			atomic.AddInt32(&p.failedTasks, 1)
		} else {
			atomic.AddInt32(&p.completedTasks, 1)
		}
		return err
	}
}

// STOP STOPS THE WORKER POOL
func (p *WorkerPool) Stop() {
	p.shutdownOnce.Do(func() {
		p.mu.Lock()
		p.isShutdown = true
		p.mu.Unlock()

		// CANCEL CONTEXT
		p.cancel()

		// CLOSE CHANNEL
		close(p.tasks)
	})
}

// WAIT WAITS FOR ALL TASKS TO COMPLETE
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// WAITWITHTIMEOUT WAITS FOR ALL TASKS TO COMPLETE WITH A TIMEOUT
func (p *WorkerPool) WaitWithTimeout(timeout time.Duration) bool {
	// CREATE DONE CHANNEL
	done := make(chan struct{})

	// WAIT IN GOROUTINE
	go func() {
		p.wg.Wait()
		close(done)
	}()

	// WAIT WITH TIMEOUT
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// GETSTATS RETURNS THE CURRENT WORKER POOL STATISTICS
func (p *WorkerPool) GetStats() WorkerPoolStats {
	return WorkerPoolStats{
		Size:           p.size,
		RunningTasks:   int(atomic.LoadInt32(&p.runningTasks)),
		CompletedTasks: int(atomic.LoadInt32(&p.completedTasks)),
		FailedTasks:    int(atomic.LoadInt32(&p.failedTasks)),
		IsShutdown:     p.isShutdown,
	}
}

// WORKERPOOLSTATS CONTAINS STATISTICS ABOUT A WORKER POOL
type WorkerPoolStats struct {
	Size           int
	RunningTasks   int
	CompletedTasks int
	FailedTasks    int
	IsShutdown     bool
}

// PREDEFINED ERRORS
var (
	ErrPoolShutdown = &customError{msg: "worker pool is shutdown"}
)

// CUSTOMERROR IMPLEMENTS THE ERROR INTERFACE
type customError struct {
	msg string
}

// ERROR RETURNS THE ERROR MESSAGE
func (e *customError) Error() string {
	return e.msg
}
