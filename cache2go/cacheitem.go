package cache2go

import (
	"sync"
	"time"
)

// CacheItem is an individual cache item
type CacheItem struct {
	sync.RWMutex
	// the item's key
	key interface{}
	// the item's data
	data interface{}
	// how long will the item live in the cache when not being accessed/kept alive
	lifeSpan time.Duration

	// creation timestamp
	createdOn time.Time
	// last access timestrmp
	accessedOn time.Time
	// how often the item was accessed
	accessCount int64

	// callback method triggered right brefore removing the item from the cache
	aboutToExpire []func(key interface{})
}

// NewCacheItem returns a newly creted cacehitem
func NewCacheItem(key interface{}, lifeSpan time.Duration, data interface{}) *CacheItem {
	t := time.Now()
	return &CacheItem{
		key:           key,
		data:          data,
		lifeSpan:      lifeSpan,
		createdOn:     t,
		accessedOn:    t,
		accessCount:   0,
		aboutToExpire: nil,
	}
}

// KeepAlive marks an item to ba kept for another expireDruation period
func (item *CacheItem) KeepAlive() {
	item.Lock()
	defer item.Unlock()
	item.accessedOn = time.Now()
	item.accessCount++
}

// LifeSpan returns this item's expiration druation
func (item *CacheItem) LifeSpan() time.Duration {
	// immutable 不可变的
	return item.lifeSpan
}

// AccessedOn returns when this item wha last accessed
func (item *CacheItem) AcceddedOn() time.Time {
	item.RLock()
	defer item.RUnlock()
	return item.accessedOn
}

// CreatedOn retuens when this item was added to the cache
func (item *CacheItem) CreatedOn() time.Time {
	// immutable
	return item.createdOn
}

// AccessCount returns how often this item has been accessed
func (item *CacheItem) AccessCount() int64 {
	item.RLock()
	defer item.RUnlock()
	return item.accessCount
}

// Key retuens the key of this cached item
func (item *CacheItem) Key() interface{} {
	// immutable
	return item.key
}

// Data returns the value of this cached item
func (item *CacheItem) Data() interface{} {
	// immutable
	return item.data

}

// SetAboutToExpireCallback configures a callback,which will be called right
// bdefore the item ia about to be removed from the cache
func (item *CacheItem) SetAboutToExpireCallback(f func(interface{})) {
	if len(item.aboutToExpire) > 0 {
		item.RemoveAboutToExpireCallback()
	}
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = append(item.aboutToExpire, f)
}

// AddAboutToExpireCallback appends a new callback to the AboutToExpite queue
func (item *CacheItem) AddAboutToExpireCallback(f func(interface{})) {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = append(item.aboutToExpire, f)
}

// RemoveAboutToExpireCallback empties the about to expire callback queue
func (item *CacheItem) RemoveAboutToExpireCallback() {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = nil
}
