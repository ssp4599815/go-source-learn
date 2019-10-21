package lru

import "container/list"

// Cache结构用于实现LRU cache算法；并发访问不安全
type Cache struct {
	// 最大入口数，也就是缓存中最多存几条数据，超过了就触发数据淘汰；0表示没有限制
	MaxEntries int
	// 销毁前回调
	onEvicted func(key Key, value interface{})
	// 链表
	ll *list.List
	// key为任意类型，值为指向链表一个结点的指针
	cache map[interface{}]*list.Element
}

// 任意可比较类型
type Key interface{}

type entry struct {
	key   Key
	value interface{}
}

func New(maxEntries int) *Cache {
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

func (c *Cache)Add(key key, value interface{})  {

}
