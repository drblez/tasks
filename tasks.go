package tasks

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Logger interface for Tasks dispatcher
type Logger interface {
	// Debugf out debug information
	Debugf(format string, args ...interface{})
	// Errorf out error information
	Errorf(format string, args ...interface{})
	// WithField add additional field to Debugf/Errorf information
	AddField(ket string, value interface{}) Logger
}

// "Do nothing" implementation of Logger interface
type NullLogger struct{}

// "Do nothing" Debugf
func (NullLogger) Debugf(format string, args ...interface{}) {
	// do nothing
}

// "Do nothing" Errorf
func (NullLogger) Errorf(format string, args ...interface{}) {
	// do nothing
}

// "Do nothing" WithField
func (NullLogger) AddField(key string, value interface{}) Logger {
	return &NullLogger{}
}

// Simple implementation of Logger interface
type SimpleLogger struct {
	parent *SimpleLogger
	key    string
	value  interface{}
}

func (logger *SimpleLogger) makeFieldsString() string {
	strs := strings.Builder{}
	if logger.parent != nil {
		strs.WriteString(fmt.Sprintf("[%s: %v] ", logger.key, logger.value))
		strs.WriteString(logger.parent.makeFieldsString())
	}
	return strs.String()
}

func (logger *SimpleLogger) makeString(format string, args ...interface{}) string {
	strs := strings.Builder{}
	strs.WriteString(logger.makeFieldsString())
	strs.WriteString(fmt.Sprintf(format, args...))
	return strs.String()
}

// Simple Debugf
func (logger *SimpleLogger) Debugf(format string, args ...interface{}) {
	log.Printf("%s [DEBUG] %s", time.Now().Format(time.RFC3339), logger.makeString(format, args...))
}

// Simple Errorf
func (logger *SimpleLogger) Errorf(format string, args ...interface{}) {
	log.Printf("%s [ERROR] %s", time.Now().Format(time.RFC3339), logger.makeString(format, args...))
}

// Simple WithField
func (logger *SimpleLogger) AddField(key string, value interface{}) Logger {
	return &SimpleLogger{
		parent: logger,
		key:    key,
		value:  value,
	}
}

type nullableWaitGroup struct {
	wg *sync.WaitGroup
}

func (nwg *nullableWaitGroup) Add() {
	if nwg.wg != nil {
		nwg.wg.Add(1)
	}
}

func (nwg *nullableWaitGroup) Done() {
	if nwg.wg != nil {
		nwg.wg.Done()
	}
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

func (w worker) start(ctx context.Context, nwg *nullableWaitGroup) {
	w.log.Debugf("Starting worker...")
	nwg.Add()
	go func() {
		defer nwg.Done()
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

// NewDispatcher create new dispatcher with maxWorkers workers and
// payload queue with payloadQueueSize size
func NewDispatcher(maxWorkers int, payloadQueueSize int, log Logger) *Dispatcher {
	pool := make(chan chan job, maxWorkers)
	d := &Dispatcher{
		pool:         pool,
		payloadQueue: make(chan job, payloadQueueSize),
		maxWorkers:   maxWorkers,
	}
	if log == nil {
		d.log = &NullLogger{}
	} else {
		d.log = log
	}
	return d
}

// Run create workers and start dispatcher
// use context.WithCancel(...) to cancel workers and dispatcher
// or context.Background() to prevent cancelling dispatcher
// and sync.WaitGroup for waiting workers done or nil to stop working without waiting
// when main routine end
func (d *Dispatcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	nwg := &nullableWaitGroup{
		wg: wg,
	}
	d.log.Debugf("Starting workers...")
	for i := 0; i < d.maxWorkers; i++ {
		workerCtx, _ := context.WithCancel(ctx)
		worker := newWorker(d.pool, d.log.AddField("worker", i))
		worker.start(workerCtx, nwg)
	}
	nwg.Add()
	go d.dispatch(ctx, nwg)
}

// Payload send payload to payload queue
func (d *Dispatcher) Payload(payloadFunc func() error) {
	d.payloadQueue <- job{Payload: payloadFunc}
}

func (d *Dispatcher) dispatch(ctx context.Context, nwg *nullableWaitGroup) {
	defer nwg.Done()
	d.log.Debugf("Starting dispatching...")
	for {
		select {
		case j := <-d.payloadQueue:
			nwg.Add()
			go func(j job) {
				defer nwg.Done()
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
