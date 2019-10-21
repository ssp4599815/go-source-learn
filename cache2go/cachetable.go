package cache2go

import (
	"log"
	"sort"
	"sync"
	"time"
)

// CacheTable is a table within the cache

type CacheTable struct {
	sync.RWMutex
	// the table's name
	name string
	// all cached items
	items map[interface{}]*CacheItem

	// Timer responsible for trigger cleanup
	cleanupTimer *time.Timer
	// Current timer duration
	cleanupInterval time.Duration

	// the logger used for this table
	logger *log.Logger

	// callbak method triggered when trying to load a non-existiog key
	loadData func(key interface{}, args ...interface{}) *CacheItem
	// callback method triggered when adding a new item to the cache
	addedItem []func(item *CacheItem)
	// callback method triggered before deleting an item from the cache
	aboutToDeleteItem []func(item *CacheItem)
}

// count retouns how many items are curengyl stored in the cache
func (table *CacheTable) Count() int {
	table.RLock()
	defer table.RUnlock()
	return len(table.items)
}

// Foreach all items
func (table *CacheTable) Foreach(trans func(key interface{}, item *CacheItem)) {
	table.RLock()
	defer table.RUnlock()

	for k, v := range table.items {
		trans(k, v)
	}
}

// SetDataLoder configures a data-loader callback which will be called when trying to access a non-existing key,
// the key and -...n additional arguments are passed to the cacllback function
func (table *CacheTable) SetDataLoader(f func(interface{}, ...interface{}) *CacheItem) {
	table.Lock()
	defer table.Unlock()
	table.loadData = f
}

// SetAddedItemCallback configure a callback which will be called evenry time a new item is added to the cache
func (table *CacheTable) SetAddedItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.addedItem = f
}

// SetAboutToDeleteItemCallback configures a callback which will be called every time an item is about to be removed from cache
func (table *CacheTable) SetAboutToDeleteItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = f
}

// SetLogger sets the logger to be used by this cache table
func (table *CacheTable) SetLogger(logger *log.Logger) {
	table.Lock()
	defer table.Unlock()
	table.logger = logger
}

// Expiration check loop, triggered by a self-adjusting timer
func (table *CacheTable) expirationCheck() {
	table.Lock()
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
	if table.cleanupInterval > 0 {
		table.log("exporation check tirggered after", table.cleanupInterval, "for table", table.name)
	} else {
		table.log("expiration check installed for table", table.name)
	}
	// to be more accurate with timers ,wo would need to update 'now' on every
	// loop iteration, not sure it's really efficient though
	now := time.Now()
	smallestDuration := 0 * time.Second
	for key, item := range table.items {
		// cache values so wo don't keep blocking the mutex
		item.RLock()
		lifeSpan := item.lifeSpan
		accessedOn := item.accessedOn
		item.RUnlock()

		if lifeSpan == 0 {
			continue
		}
		if now.Sub(accessedOn) >= lifeSpan {
			// item has excessed its lifespan
			table.deleteInternal(key)
		} else {
			// find the item chronologically closest to its end-of-lifespan
			if smallestDuration == 0 || lifeSpan-now.Sub(accessedOn) < smallestDuration {
				smallestDuration = lifeSpan - now.Sub(accessedOn)
			}
		}
	}

	// setup the interval for the next cleanup run
	table.cleanupInterval = smallestDuration
	if smallestDuration > 0 {
		table.cleanupTimer = time.AfterFunc(smallestDuration, func() {
			go table.expirationCheck()
		})
	}
	table.Unlock()
}

func (table *CacheTable) addInternal(item *CacheItem) {
	// careful: do not run this method unless the table-mutex is locked
	// it will unlock it for the caller before running the callbacks and checks
	table.log("adding item with key", item.key, "and lifespanof", item.lifeSpan, "to table", table.name)
	table.items[item.key] = item

	// cache values so wo don't keep blocking the mutex
	expDur := table.cleanupInterval
	addedItem := table.addedItem
	table.Unlock()

	// trigger callback after adding an item to cache
	if addedItem := nil {
		for _, callback := range addedItem {
			callback(item)
		}
	}

	// if wo haven't set up any expiration check timer or found a more imminent item
	if item.lifeSpan > 0 && (expDur == 0 || item.lifeSpan < expDur) {
		table.expirationCheck()
	}
}

// Add adds a key/value pair to the cahce
// paramter key is the item's cache-key
// parameter lifespan determines after which time period without an access the item
// will get removed from the cache
// parameter data is the item's value
func (table *CacheTable) Add(key interface{}, lifeSpan time.Duration, data interface{}) *CacheItem {
	item := NewCacheItem(key, lifeSpan, data)

	// add item to cache
	table.Lock()
	table.addInternal(item)
	return item
}

func (table *CacheTable) deleteInternal(key interface{}) (*CacheItem, error) {
	r, ok := table.items[key]
	if !ok {
		return nil, ErrKeyNotFount
	}
	// cache value so we don't keep blocking the mutex
	aboutToDeleteItem := table.aboutToDeleteItem
	table.Unlock()

	// trigger callbacks before deleting an item from cache
	if aboutToDeleteItem != nil {
		for _, callback := range aboutToDeleteItem {
			callback(r)
		}
	}
	r.RLock()
	defer r.Unlock()

	if r.aboutToExpire != nil {
		for _, callback := range r.aboutToExpire {
			callback(key)
		}
	}

	table.Lock()
	table.log("Deleting item with key", key, "created on", r.createdOn, "and hit", r.accessCount, "time for table", table.name)
	delete(table.items, key)

	return r, nil
}

// Delete an item from the cache
func (table *CacheTable) Delete(key interface{}) (*CacheItem, error) {
	table.Lock()
	defer table.Unlock()

	return table.deleteInternal(key)
}

// exists returns whether an item exists in the cache , unlike the value method
// exists nrither tries to fetch data  via the loaddata callback nor does it
// keep the item alive in the cache
func (table *CacheTable) Exists(key interface{}) bool {
	table.RLock()
	defer table.Unlock()
	_, ok := table.items[key]
	return ok
}

// NotFoundAdd check whether an item is not yet cached unlike the exists
// method this also adds data if the key could not be found
func (table *CacheTable) NotFoundAdd(key interface{}, lifeSpan time.Duration, data interface{}) bool {
	table.Lock()

	if _, ok := table.items[key]; ok {
		table.Unlock()
		return false
	}
	item := NewCacheItem(key, lifeSpan, data)
	table.addInternal(item)
	return true
}

// value returns an item from the cache and marks it to be kept alive , you can
// pass additional arguments to your dataloader callback function
func (table *CacheTable) Value(key interface{}, args ...interface{}) (*CacheItem, error) {
	table.RLock()
	r, ok := table.items[key]
	loadData := table.loadData
	table.RUnlock()

	if ok {
		// updata access counter an timestamp
		r.KeepAlive()
		return r, nil
	}
	// item doesn't exist in cache ,try and fetch it with a data-loader
	if loadData != nil {
		item := loadData(key, args...)
		if item != nil {
			table.Add(key, item.lifeSpan, item.data)
			return item, nil
		}
		return nil, ErrKeyNotFoundOrLoadable
	}
	return nil, ErrKeyNotFount
}

// flush deletes all items from this cache table
func (table *CacheTable) Flush() {
	table.Lock()
	defer table.Unlock()

	table.log("Flushing table", table.name)

	table.items = make(map[interface{}]*CacheItem)
	table.cleanupInterval = 0
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
}

// CacheItemPair maps key to access conter
type CacheItemPair struct {
	Key         interface{}
	AccessCount int64
}

// cacheitempairlist is a slice of cacheitempairs that implements sort
// interface to sort by accesscount

type CacheItemPairList []CacheItemPair

func (p CacheItemPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p CacheItemPairList) Len() int           { return len(p) }
func (p CacheItemPairList) Less(i, j int) bool { return p[i].AccessCount > p[j].AccessCount }

// MostAccessed returns the most accessed items in this cache table
func (table *CacheTable) MostAccessed(count int64) []*CacheItem {
	table.RLock()
	defer table.RUnlock()
	p := make(CacheItemPairList, len(table.items))
	i := 0
	for k, v := range table.items {
		p[i] = CacheItemPair{k, v.accessCount}
		i++
	}
	sort.Sort(p)

	var r []*CacheItem
	c := int64(0)
	for _, v := range p {
		if c >= count {
			break
		}

		item, ok := table.items[v.Key]
		if ok {
			r = append(r, item)
		}
		c++
	}
	return r
}

func (table *CacheTable) log(v ...interface{}) {
	if table.logger == nil {
		return
	}
	table.logger.Println(v...)
}
