package main

import (
	"fmt"
	"github.com/muesli/cache2go"
	"strconv"
)

func main() {
	cache := cache2go.Cache("myCache")

	cache.SetDataLoader(func(key interface{}, args ...interface{}) *cache2go.CacheItem {
		val := "this is a test with key " + key.(string)

		item := cache2go.NewCacheItem(key, 0, val)
		return item
	})
	for i := 0; i < 10; i++ {
		res, err := cache.Value("someKey_" + strconv.Itoa(i))
		if err != nil {
			fmt.Println("error retrieving value from cache", err)
		}
		fmt.Println("found value in cache:", res.Data())
	}
}
