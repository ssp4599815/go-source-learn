package main

import (
	"fmt"
	"github.com/muesli/cache2go"
	"time"
)

type myStruct struct {
	text     string
	moreData []byte
}

func main() {
	cache := cache2go.Cache("myCache")

	val := myStruct{"this is a test", []byte{}}
	cache.Add("someKey", 5*time.Second, &val)

	res, err := cache.Value("someKey")
	if err != nil {
		fmt.Println("error retrieving value from cache:", err)
	}
	fmt.Println("found value in cache:", res.Data().(*myStruct).text)
	time.Sleep(6 * time.Second)
	res, err = cache.Value("someKey")
	if err != nil {
		fmt.Println("Item is not cached (anymore)")
	}
	cache.Add("someKey", 0, &val)

	cache.SetAboutToDeleteItemCallback(func(item *cache2go.CacheItem) {
		fmt.Println("deleteing:", item.Key(), item.Data().(*myStruct).text, item.CreatedOn())
	})
	cache.Delete("someKey")

	cache.Flush()
}
