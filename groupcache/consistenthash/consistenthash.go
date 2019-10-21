package consistenthash

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
)

// 一个函数类型
type Hash func(data []byte) uint32

type Map struct {
	hash     Hash
	replicas int // 每个key 的副本数量
	// key 为 哈希环上的一个点
	keys []int // sorted
	// 哈希环上的一个 点到服务器名的映射
	hashMap map[int]string
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		// ChecksumIEEE 计算 []byte 的校验和，返回uint32类型结果
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (m *Map) IsEmpty() bool {
	return len(m.keys) == 0
}

// Adds some keys to the hash
// 将缓存服务器加到Map中，比如Cache A、Cache B作为keys，
// 如果副本数指定的是2，那么Map中存的数据是Cache A#1、Cache A#2、Cache B#1、Cache B#2的hash结果
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	fmt.Println(m)
	// 升序排列这个 int切片
	sort.Ints(m.keys)
}

// Gets the closest item in the hash to the provided key
func (m *Map) Get(key string) string {
	if m.IsEmpty() {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	fmt.Println(hash)
	// Binary search for appropriate replics
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	fmt.Println(idx)
	if idx == len(m.keys) {
		idx = 0
	}
	return m.hashMap[m.keys[idx]]
}
