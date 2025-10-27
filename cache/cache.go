package cache

import (
	"crypto/sha256"
	"fmt"
	"llm-language-server/lsp"
	"strings"
	"time"
)

type CacheValue struct {
	Data []lsp.CompletionItem
	Ex   time.Time
}

var cacheMap = make(map[string]CacheValue)
var EX = 60 * time.Second

func checkExpiredKeys() {
	for key, value := range cacheMap {
		if time.Now().After(value.Ex) {
			delete(cacheMap, key)
		}
	}
}

func getKey(prompt string, suffix string) string {
	cacheKey := fmt.Sprintf("%s<FIM_MIDDLE>%s", prompt, suffix)
	cacheKeyHash := sha256.New()
	cacheKeyHash.Write([]byte(cacheKey))
	return string(cacheKeyHash.Sum(nil))
}

func Set(prompt string, suffix string, value []lsp.CompletionItem) {
	key := getKey(prompt, suffix)
	cacheMap[key] = CacheValue{Data: value, Ex: time.Now().Add(EX)}
	for _, item := range value {
		chars := strings.Split(item.InsertText, "")
		pendingLine := ""
		for _, char := range chars {
			pendingLine += char
			item.InsertText = item.InsertText[1:]
			pendingKey := getKey(prompt+pendingLine, suffix)
			cacheMap[pendingKey] = CacheValue{Data: []lsp.CompletionItem{item}, Ex: time.Now().Add(EX)}
		}
	}
}

func Get(prompt string, suffix string) ([]lsp.CompletionItem, bool) {
	key := getKey(prompt, suffix)
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

func Reset() {
	cacheMap = make(map[string]CacheValue)
}
