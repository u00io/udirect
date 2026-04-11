package stats

import "sync"

type Stats struct {
	Data map[string]int64
}

func init() {
	Instance.Data = make(map[string]int64)
}

var mtx sync.Mutex
var Instance Stats

func Get() Stats {
	var result Stats
	mtx.Lock()
	result.Data = make(map[string]int64)
	for k, v := range Instance.Data {
		result.Data[k] = v
	}
	mtx.Unlock()
	return result
}

func Set(key string, value int64) {
	mtx.Lock()
	Instance.Data[key] = value
	mtx.Unlock()
}

func Inc(key string) {
	mtx.Lock()
	Instance.Data[key]++
	mtx.Unlock()
}

func Dec(key string) {
	mtx.Lock()
	Instance.Data[key]--
	mtx.Unlock()
}
