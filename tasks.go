package tasks

import (
	"context"
	"sync"
)

type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	WithField(ket string, value interface{}) Logger
}

// job Задание, которое надо выполнить
type job struct {
	Payload func() error
}

// worker Исполнитель
type worker struct {
	Pool           chan chan job
	PayloadChannel chan job
	log            Logger
}

func newWorker(workerPool chan chan job, log Logger) worker {
	return worker{
		Pool:           workerPool,
		PayloadChannel: make(chan job),
		log:            log,
	}
}

func (w worker) start(ctx context.Context, wg *sync.WaitGroup) {
	w.log.Debugf("Starting worker...")
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// При первом запуске: зарегистрируем нового исполнителя в пуле инсполнителей
			// При последующтх выполнениях цикла: отметимся в пуле, что мы еще живы
			w.Pool <- w.PayloadChannel

			select {
			case job := <-w.PayloadChannel:
				// Пришло задание
				w.log.Debugf("Do payload...")
				if err := job.Payload(); err != nil {
					w.log.Errorf("Do payload error: %+v", err)
				}
				w.log.Debugf("done...")
			case <-ctx.Done():
				w.log.Debugf("Shutdown worker...")
				return
			}
		}
	}()
}

type Dispatcher struct {
	pool         chan chan job
	payloadQueue chan job
	maxWorkers   int
	log          Logger
}

func NewDispatcherDefault(log Logger) *Dispatcher {
	return NewDispatcher(10, 10000, log)
}

func NewDispatcher(maxWorkers int, payloadQueueSize int, log Logger) *Dispatcher {
	pool := make(chan chan job, maxWorkers)
	return &Dispatcher{
		pool:         pool,
		payloadQueue: make(chan job, payloadQueueSize),
		maxWorkers:   maxWorkers,
		log:          log,
	}
}

func (d *Dispatcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	d.log.Debugf("Starting workers...")
	for i := 0; i < d.maxWorkers; i++ {
		workerCtx, _ := context.WithCancel(ctx)
		worker := newWorker(d.pool, d.log.WithField("worker", i))
		worker.start(workerCtx, wg)
	}
	wg.Add(1)
	go d.dispatch(ctx, wg)
}

func (d *Dispatcher) Payload(payloadFunc func() error) {
	d.payloadQueue <- job{Payload: payloadFunc}
}

func (d *Dispatcher) dispatch(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	d.log.Debugf("Starting dispatching...")
	for {
		select {
		case j := <-d.payloadQueue:
			// a job request has been received
			wg.Add(1)
			go func(j job) {
				defer wg.Done()
				// попробуем получить инсполнителя из пула всех исполнителей
				jobChannel := <-d.pool
				// отправим задание исполнителю
				jobChannel <- j
			}(j)
		case <-ctx.Done():
			d.log.Debugf("Shutdown dispatcher...")
			return
		}
	}
}
