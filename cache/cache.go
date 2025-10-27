package cache

import (
	"crypto/sha256"
	"fmt"
	"llm-language-server/lsp"
	"time"
)

type CacheValue struct {
	Data []lsp.CompletionItem
	Ex   time.Time
}

var cacheMap = make(map[string]CacheValue)
var EX = 5 * time.Second

func checkExpiredKeys() {
	for key, value := range cacheMap {
		if time.Now().After(value.Ex) {
			delete(cacheMap, key)
		}
	}
}

func GetKey(prompt string, suffix string) string {
	cacheKey := fmt.Sprintf("%s<FIM_MIDDLE>%s", prompt, suffix)
	cacheKeyHash := sha256.New()
	cacheKeyHash.Write([]byte(cacheKey))
	return string(cacheKeyHash.Sum(nil))
}

func Set(key string, value []lsp.CompletionItem) {
	cacheMap[key] = CacheValue{Data: value, Ex: time.Now().Add(EX)}
}

func Get(key string) ([]lsp.CompletionItem, bool) {
	value, exists := cacheMap[key]

	if exists {
		return value.Data, exists
	} else {
		return nil, exists
	}
}

var ticker *time.Ticker

func Init() {
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
