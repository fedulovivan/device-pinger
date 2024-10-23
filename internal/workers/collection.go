package workers

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/fedulovivan/device-pinger/internal/counters"
)

var (
	collectionWg   sync.WaitGroup
	collection     map[string](*Worker)
	collectionLock sync.RWMutex
)

func add_unsafe(worker *Worker) {
	if collection == nil {
		collection = make(map[string]*Worker)
	}
	collection[worker.target] = worker
	slog.Debug(tagBase.F("Worker added"), "len", len_unsafe())
}

func Get(target string) (*Worker, error) {
	collectionLock.RLock()
	defer collectionLock.RUnlock()
	return get_unsafe(target)
}

func get_unsafe(target string) (*Worker, error) {
	w, ok := collection[target]
	if !ok {
		return nil, errors.New("not exist")
	}
	return w, nil
}

func StopAll() {
	for _, worker := range collection {
		go worker.Stop()
	}
}

func Create(
	target string,
	onStatusChange OnlineStatusChangeHandler,
) (*Worker, error) {
	collectionLock.Lock()
	defer collectionLock.Unlock()
	_, ok := collection[target]
	if ok {
		return nil, errors.New("already exist")
	}
	w, _ := New(target, onStatusChange)
	collectionWg.Add(1)
	add_unsafe(w)
	counters.Workers.Inc()
	return w, nil
}

func Delete(target string, onChange OnlineStatusChangeHandler) error {
	collectionLock.Lock()
	defer collectionLock.Unlock()
	worker, err := get_unsafe(target)
	if err != nil {
		return err
	}
	worker.Stop()
	delete(collection, target)
	slog.Debug(tagBase.F("Worker deleted"), "size", len_unsafe())
	counters.Workers.Dec()
	return nil
}

func Wait() {
	collectionWg.Wait()
}

func len_unsafe() int {
	return len(collection)
}

func Len() int {
	collectionLock.RLock()
	defer collectionLock.RUnlock()
	return len_unsafe()
}
