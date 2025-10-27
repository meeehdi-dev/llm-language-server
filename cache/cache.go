package cache

import (
	"llm-language-server/lsp"
	"time"
)

type CacheValue struct {
	Data []lsp.CompletionItem
	Ex   time.Time
}

var cacheMap map[string]CacheValue
var started = false

var ex = 60 * time.Second

func checkExpiredKeys() {
	for key, value := range cacheMap {
		if time.Now().After(value.Ex) {
			delete(cacheMap, key)
		}
	}
}

func Set(prompt string, suffix string, value []lsp.CompletionItem) {
	if !started {
		return
	}

	cacheMap[prompt+suffix] = CacheValue{Data: value, Ex: time.Now().Add(ex)}
}

func Get(prompt string, suffix string) ([]lsp.CompletionItem, bool) {
	if !started {
		return nil, false
	}

	value, exists := cacheMap[prompt+suffix]

	if exists {
		return value.Data, true
	} else {
		return nil, false
	}
}

func Reset() {
	cacheMap = make(map[string]CacheValue)
}

var ticker *time.Ticker

func Init() {
	Reset()
	started = true

	ticker = time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkExpiredKeys()
		}
	}
}

func Shutdown() {
	if ticker != nil {
		ticker.Stop()
	}
}
