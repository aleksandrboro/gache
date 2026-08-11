package storage

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"time"
)

type Store struct {
	shards [256]Shard
}

type Shard struct {
	mu   sync.RWMutex
	data map[string]*Entry
}

type Entry struct {
	Value    []byte
	ExpireAt int64
}

func NewStore() *Store {
	store := &Store{}
	for i := range store.shards {
		store.shards[i].data = make(map[string]*Entry)
	}

	return store
}

func (s *Store) Get(key string) ([]byte, bool) {
	sh := s.getShard(key)
	sh.mu.RLock()
	v, ok := sh.data[key]
	if !ok {
		sh.mu.RUnlock()
		return nil, ok
	}

	if sh.data[key].ExpireAt > 0 && time.Now().UnixNano() > sh.data[key].ExpireAt {
		sh.mu.RUnlock()
		sh.mu.Lock()
		entry := sh.data[key]
		if entry != nil && entry.ExpireAt > 0 && time.Now().UnixNano() > entry.ExpireAt {
			delete(sh.data, key)
		}

		sh.mu.Unlock()
		return nil, false
	}

	sh.mu.RUnlock()
	return v.Value, ok
}

func (s *Store) Set(key string, value []byte) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.data[key] = &Entry{
		Value: value,
	}
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

	v, ok := shard.data[key]
	if !ok {
		shard.data[key] = &Entry{}
	}
	var num int64

	if v == nil {
		num = 0
	} else {
		var err error
		num, err = strconv.ParseInt(string(v.Value), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("ERR value is not an integer or out of range")
		}
	}

	if (increment > 0 && num > math.MaxInt64-increment) ||
		(increment < 0 && num < math.MinInt64-increment) {
		return 0, fmt.Errorf("ERR increment or decrement would overflow")
	}

	num += increment

	shard.data[key].Value = []byte(strconv.FormatInt(num, 10))

	return num, nil
}

func (s *Store) MSet(pairs map[string][]byte) {
	shards := []*Shard{}

	for k := range pairs {
		shard := s.getShard(k)
		if !slices.Contains(shards, shard) {
			shards = append(shards, shard)
		}
	}

	for _, shard := range shards {
		shard.mu.Lock()
	}

	for k, v := range pairs {
		shard := s.getShard(k)
		shard.data[k] = &Entry{
			Value: v,
		}
	}

	for _, shard := range shards {
		shard.mu.Unlock()
	}
}

func (s *Store) MGet(keys []string) [][]byte {
	resp := make([][]byte, len(keys))

	for i, key := range keys {
		shard := s.getShard(key)
		shard.mu.RLock()
		v, ok := shard.data[key]
		if !ok {
			resp[i] = nil
		} else {
			resp[i] = v.Value
		}
		shard.mu.RUnlock()
	}

	return resp
}

func (s *Store) SetWithTTL(key string, value []byte, ttl int64) {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.data[key] = &Entry{
		Value:    value,
		ExpireAt: time.Now().Add(time.Duration(ttl)).UnixNano(),
	}
}

func (s *Store) Expire(key string, ttl int64) bool {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.data[key]; !ok {
		return false
	}

	shard.data[key].ExpireAt = time.Now().Add(time.Duration(ttl)).UnixNano()

	return true
}

func (s *Store) TTL(key string) int64 {
	shard := s.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	v, ok := shard.data[key]
	if !ok {
		return -2
	}

	if v.ExpireAt == 0 {
		return -1
	}

	return v.ExpireAt - time.Now().UnixNano()
}

func (s *Store) Persist(key string) bool {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.data[key]; !ok {
		return false
	}

	if shard.data[key].ExpireAt == 0 {
		return false
	}

	shard.data[key].ExpireAt = 0
	return true
}
