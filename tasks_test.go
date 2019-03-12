package tasks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	dispatcher := NewDispatcher(2, 10, &SimpleLogger{})
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	dispatcher.Run(ctx, wg)
	f := func(i int) func() error {
		return func() error {
			fmt.Printf("Hi, I am function #%d\n", i)
			return nil
		}
	}
	i := 0
	for ; i < 20; i++ {
		dispatcher.Payload(f(i))
	}
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
}

func Test2(t *testing.T) {
	dispatcher := NewDispatcher(2, 10, nil)
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	dispatcher.Run(ctx, wg)
	f := func(i int) func() error {
		return func() error {
			fmt.Printf("Hi, I am function #%d\n", i)
			return nil
		}
	}
	i := 0
	for ; i < 2; i++ {
		dispatcher.Payload(f(i))
	}
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
}

/*

type TasksLogger struct {
	*logrus.Entry
}

func NewTaskLogger() *TasksLogger {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.Formatter = &logrus.TextFormatter{}

	tl := &TasksLogger{}
	tl.Entry = log.WithContext(context.Background())
	return tl
}

func (logger *TasksLogger) AddField(key string, value interface{}) Logger {
	tl := &TasksLogger{}
	tl.Entry = logger.WithField(key, value)
	return tl
}

func Test3(t *testing.T) {
	dispatcher := NewDispatcher(2, 10, NewTaskLogger())
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	dispatcher.Run(ctx, wg)
	f := func(i int) func() error {
		return func() error {
			fmt.Printf("Hi, I am function #%d\n", i)
			return nil
		}
	}
	i := 0
	for ; i < 20; i++ {
		dispatcher.Payload(f(i))
	}
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
}

*/
