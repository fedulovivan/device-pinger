package workers

import (
	"log/slog"
	"sync"

	"github.com/fedulovivan/device-pinger/lib/utils"
)

var (
	Wg     sync.WaitGroup
	Errors chan error

	workers     map[string](*Worker)
	workersLock sync.RWMutex
)

func Add(worker *Worker) {
	workersLock.Lock()
	defer workersLock.Unlock()
	if workers == nil {
		workers = make(map[string]*Worker)
	}
	workers[worker.target] = worker
	slog.Debug("[MAIN] Worker added", "size", GetCount())
}

// not nil-protected, use Has in outer code before calling Get
func Get(target string) *Worker {
	workersLock.RLock()
	defer workersLock.RUnlock()
	return workers[target]
}

func Has(target string) bool {
	workersLock.RLock()
	defer workersLock.RUnlock()
	_, ok := workers[target]
	return ok
}

func Delete(target string, onChange OnlineStatusChangeHandler) {
	workersLock.Lock()
	defer workersLock.Unlock()
	w, ok := workers[target]
	if ok {
		w.Stop(onChange)
		delete(workers, target)
		slog.Debug("[MAIN] Worker deleted", "size", GetCount())
	}
}

func GetAsList() []*Worker {
	return utils.Values(workers)
}

func GetCount() int {
	return len(workers)
}
