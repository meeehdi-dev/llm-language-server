package cache

import (
	"container/list"
	"llm-language-server/lsp"
	"strings"
	"time"
)

type Cache struct {
	Prompt string
	Suffix string
	Data   []lsp.CompletionItem
	Ex     time.Time
}

var cache *list.List
var started = false

var ex = 600 * time.Second

func checkExpiredKeys() {
	for e := cache.Front(); e != nil; e = e.Next() {
		value := e.Value.(Cache)
		if time.Now().After(value.Ex) {
			cache.Remove(e)
		}
	}
}

func Set(prompt string, suffix string, value []lsp.CompletionItem) {
	if !started {
		return
	}

	cache.PushBack(Cache{Data: value, Prompt: prompt, Suffix: suffix, Ex: time.Now().Add(ex)})
}

func Get(prompt string, suffix string) ([]lsp.CompletionItem, bool) {
	if !started {
		return nil, false
	}

	for e := cache.Front(); e != nil; {
		next := e.Next()
		value := e.Value.(Cache)
		for _, item := range value.Data {
			if strings.HasPrefix(value.Prompt+item.InsertText, prompt) && strings.HasSuffix(suffix, value.Suffix) {
				clonedData := make([]lsp.CompletionItem, len(value.Data))
				for i, item := range value.Data {
					clonedItem := item
					clonedItem.InsertText = item.InsertText[len(prompt)-len(value.Prompt):]
					clonedData[i] = clonedItem
				}
				return clonedData, true
			}
		}
		e = next
	}

	return []lsp.CompletionItem{}, false
}

func Reset() {
	cache = list.New()
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
