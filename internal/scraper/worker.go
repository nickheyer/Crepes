package scraper

import "sync"

// -- WORKER POOL STUFF --

// WORKER REPRESENTS A CONCURRENT TASK EXECUTOR
type Worker struct {
	tasks chan func()
	wg    *sync.WaitGroup
}

// NEWWORKER CREATES A NEW WORKER POOL
func NewWorker(maxConcurrent int) *Worker {
	w := &Worker{
		tasks: make(chan func(), 100), // BUFFER SIZE FOR QUEUED TASKS
		wg:    &sync.WaitGroup{},
	}

	// START WORKER GOROUTINES
	for range maxConcurrent {
		go func() {
			for task := range w.tasks {
				task()
				w.wg.Done()
			}
		}()
	}

	return w
}

// SUBMIT ADDS A TASK TO THE WORKER POOL
func (w *Worker) Submit(task func()) {
	w.wg.Add(1)
	w.tasks <- task
}

// WAIT BLOCKS UNTIL ALL TASKS COMPLETE
func (w *Worker) Wait() {
	w.wg.Wait()
}

// CLOSE SHUTS DOWN THE WORKER POOL
func (w *Worker) Close() {
	close(w.tasks)
}
