package cache

import (
	"crypto/sha256"
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

func getKey(prompt string, suffix string) string {
	cacheKeyHash := sha256.New()
	cacheKeyHash.Write([]byte(prompt + "_" + suffix))
	return string(cacheKeyHash.Sum(nil))
}

func Set(prompt string, suffix string, value []lsp.CompletionItem) {
	if !started {
		return
	}

	key := getKey(prompt, suffix)
	cacheMap[key] = CacheValue{Data: value, Ex: time.Now().Add(ex)}
}

func Get(prompt string, suffix string) ([]lsp.CompletionItem, bool) {
	if !started {
		return nil, false
	}

	key := getKey(prompt, suffix)
	value, exists := cacheMap[key]

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
