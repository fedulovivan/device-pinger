package workers

import (
	"errors"
	"log/slog"
	"sync"
)

type Collection struct {
	sync.RWMutex
	wg        sync.WaitGroup
	data      map[TargetAddr](*Worker)
	lenChange chan int
}

func NewCollection() *Collection {
	return &Collection{
		data:      make(map[TargetAddr]*Worker),
		lenChange: make(chan int),
	}
}

func (c *Collection) Get(target TargetAddr) (*Worker, error) {
	c.RLock()
	defer c.RUnlock()
	return c.get_unsafe(target)
}

func (c *Collection) get_unsafe(target TargetAddr) (*Worker, error) {
	w, ok := c.data[target]
	if !ok {
		return nil, errors.New("not exist")
	}
	return w, nil
}

func (c *Collection) OnLenChange() chan int {
	return c.lenChange
}

func (c *Collection) StopAll() {
	for _, worker := range c.data {
		go worker.Stop()
	}
}

func (c *Collection) Create(
	target TargetAddr,
	onStatusChange OnlineStatusChangeHandler,
) (*Worker, error) {
	c.Lock()
	defer c.Unlock()
	_, ok := c.data[target]
	if ok {
		return nil, errors.New("already exist")
	}
	c.wg.Add(1)
	worker, _ := New(target, onStatusChange)
	c.data[worker.target] = worker
	slog.Debug(tagBase.F("Worker added"), "len", len(c.data))
	c.lenChange <- len(c.data)
	go func() {
		<-worker.Done()
		c.wg.Done()
	}()
	return worker, nil
}

func (c *Collection) Delete(target TargetAddr, onChange OnlineStatusChangeHandler) error {
	c.Lock()
	defer c.Unlock()
	worker, err := c.get_unsafe(target)
	if err != nil {
		return err
	}
	worker.Stop()
	delete(c.data, target)
	slog.Debug(tagBase.F("Worker deleted"), "len", len(c.data))
	c.lenChange <- len(c.data)
	return nil
}

func (c *Collection) Wait() {
	c.wg.Wait()
}

func (c *Collection) Len() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.data)
}
