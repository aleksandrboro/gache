package storage

import (
	"fmt"
	"math"
	"strconv"
	"sync"
)

type Store struct {
	shards [256]Shard
}

type Shard struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewStore() *Store {
	store := &Store{}
	for i := range store.shards {
		store.shards[i].data = make(map[string][]byte)
	}

	return store
}

func (s *Store) Get(key string) ([]byte, bool) {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	v, ok := sh.data[key]
	return v, ok
}

func (s *Store) Set(key string, value []byte) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.data[key] = value
}

func (s *Store) Del(keys ...string) int {
	keysByShard := make(map[*Shard][]string)
	for _, key := range keys {
		sh := s.getShard(key)
		keysByShard[sh] = append(keysByShard[sh], key)
	}

	for sh := range keysByShard {
		sh.mu.Lock()
	}

	count := 0
	for sh, keysInShard := range keysByShard {
		for _, key := range keysInShard {
			if _, ok := sh.data[key]; ok {
				delete(sh.data, key)
				count++
			}
		}
	}

	for sh := range keysByShard {
		sh.mu.Unlock()
	}

	return count
}

func (s *Store) Exists(keys ...string) int {
	keysByShard := make(map[*Shard][]string)
	for _, key := range keys {
		sh := s.getShard(key)
		keysByShard[sh] = append(keysByShard[sh], key)
	}

	for sh := range keysByShard {
		sh.mu.RLock()
	}

	count := 0
	for sh, keysInShard := range keysByShard {
		for _, key := range keysInShard {
			if _, ok := sh.data[key]; ok {
				count++
			}
		}
	}

	for sh := range keysByShard {
		sh.mu.RUnlock()
	}

	return count
}

func (s *Store) Incr(key string) (int64, error) {
	return s.incrByFloat(key, 1)
}

func (s *Store) Decr(key string) (int64, error) {
	return s.incrByFloat(key, -1)
}

func (s *Store) IncrBy(key string, increment int64) (int64, error) {
	return s.incrByFloat(key, increment)
}

func (s *Store) DecrBy(key string, decrement int64) (int64, error) {
	return s.incrByFloat(key, -decrement)
}

func (s *Store) getShard(key string) *Shard {
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}

	return &s.shards[h%256]
}

func (s *Store) incrByFloat(key string, increment int64) (int64, error) {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	v := shard.data[key]
	var num int64

	if v == nil {
		num = 0
	} else {
		var err error
		num, err = strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("ERR value is not an integer or out of range")
		}
	}

	if (increment > 0 && num > math.MaxInt64-increment) ||
		(increment < 0 && num < math.MinInt64-increment) {
		return 0, fmt.Errorf("ERR increment or decrement would overflow")
	}

	num += increment

	shard.data[key] = []byte(strconv.FormatInt(num, 10))

	return num, nil
}
