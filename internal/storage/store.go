package storage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"time"
)

var (
	ErrWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	ErrEmptyList = errors.New("EMPTYLIST List is empty")
)

type Store struct {
	shards [256]Shard
}

type Shard struct {
	mu   sync.RWMutex
	data map[string]*Entry
}

type Entry struct {
	Value    Value
	ExpireAt int64
}

// Type implements [Value].
func (e *Entry) Type() string {
	panic("unimplemented")
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

	if v.Value.Type() != stringType {
		sh.mu.RUnlock()
		return nil, false
	}

	sh.mu.RUnlock()

	return v.Value.(StringValue).Data, ok
}

func (s *Store) Set(key string, value []byte) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.data[key] = &Entry{
		Value: StringValue{
			Data: value,
		},
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
				if sh.data[key].ExpireAt > time.Now().UnixNano() || sh.data[key].ExpireAt == 0 {
					count++
				}
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
		shard.data[key] = &Entry{
			Value: StringValue{},
		}

		v = shard.data[key]
	}

	if v.Value.Type() != stringType {
		return 0, ErrWrongType
	}

	var num int64

	if v.ExpireAt <= time.Now().UnixNano() && v.ExpireAt != 0 {
		num = 0
		v.ExpireAt = 0
	} else {
		var err error
		num, err = strconv.ParseInt(string(v.Value.(StringValue).Data), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("ERR value is not an integer or out of range")
		}
	}

	if (increment > 0 && num > math.MaxInt64-increment) ||
		(increment < 0 && num < math.MinInt64-increment) {
		return 0, fmt.Errorf("ERR increment or decrement would overflow")
	}

	num += increment

	shard.data[key].Value = StringValue{
		Data: []byte(strconv.FormatInt(num, 10)),
	}

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
			Value: StringValue{
				Data: v,
			},
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
			if v.ExpireAt <= time.Now().UnixNano() && v.ExpireAt != 0 {
				shard.mu.RUnlock()
				shard.mu.Lock()
				delete(shard.data, key)
				shard.mu.Unlock()
				continue
			}
			if v.Value.Type() == stringType {
				resp[i] = v.Value.(StringValue).Data
			} else {
				resp[i] = nil
			}
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
		Value: StringValue{
			Data: value,
		},
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

func (s *Store) LPush(key string, values ...[]byte) (int, error) {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.data[key]; !ok {
		shard.data[key] = &Entry{
			Value: ListValue{},
		}
	}

	if shard.data[key].Value == nil {
		shard.data[key].Value = ListValue{}
	}

	if shard.data[key].Value.Type() != listType {
		return 0, ErrWrongType
	}

	list := shard.data[key].Value.(ListValue)
	slices.Reverse(values)
	shard.data[key].Value = ListValue{
		Data: append(values, list.Data...),
	}

	return len(shard.data[key].Value.(ListValue).Data), nil
}

func (s *Store) RPush(key string, values ...[]byte) (int, error) {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.data[key]; !ok {
		shard.data[key] = &Entry{
			Value: ListValue{},
		}
	}

	if shard.data[key].Value == nil {
		shard.data[key].Value = ListValue{}
	}

	if shard.data[key].Value.Type() != listType {
		return 0, ErrWrongType
	}

	list := shard.data[key].Value.(ListValue)
	shard.data[key].Value = ListValue{
		Data: append(list.Data, values...),
	}

	return len(shard.data[key].Value.(ListValue).Data), nil
}

func (s *Store) LPop(key string) ([]byte, error) {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, ok := shard.data[key]; !ok {
		return nil, ErrEmptyList
	}

	if shard.data[key].Value == nil {
		return nil, ErrEmptyList
	}

	if shard.data[key].Value.Type() != listType {
		return nil, ErrWrongType
	}

	list := shard.data[key].Value.(ListValue)

	if len(list.Data) == 0 {
		return nil, ErrEmptyList
	}

	poped := list.Data[0]
	shard.data[key].Value = ListValue{
		Data: list.Data[1:],
	}

	if len(shard.data[key].Value.(ListValue).Data) == 0 {
		delete(shard.data, key)
	}

	return poped, nil
}

func (s *Store) RPop(key string) ([]byte, error) {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, ok := shard.data[key]; !ok {
		return nil, ErrEmptyList
	}

	if shard.data[key].Value == nil {
		return nil, ErrEmptyList
	}

	if shard.data[key].Value.Type() != listType {
		return nil, ErrWrongType
	}

	list := shard.data[key].Value.(ListValue)

	if len(list.Data) == 0 {
		return nil, ErrEmptyList
	}

	poped := list.Data[len(list.Data)-1]
	shard.data[key].Value = ListValue{
		Data: list.Data[:len(list.Data)-1],
	}

	if len(shard.data[key].Value.(ListValue).Data) == 0 {
		delete(shard.data, key)
	}

	return poped, nil
}

func (s *Store) LLen(key string) (int, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return 0, nil
	}

	if shard.data[key].Value == nil {
		return 0, nil
	}

	if shard.data[key].Value.Type() != listType {
		return 0, ErrWrongType
	}

	list := shard.data[key].Value.(ListValue)
	return len(list.Data), nil
}

func (s *Store) LRange(key string, start, stop int) ([][]byte, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return [][]byte{}, nil
	}

	if shard.data[key].Value == nil {
		return [][]byte{}, nil
	}

	if shard.data[key].Value.Type() != listType {
		return nil, ErrWrongType
	}

	list := shard.data[key].Value.(ListValue)

	if start < 0 {
		start = len(list.Data) + start
	}
	if stop < 0 {
		stop = len(list.Data) + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= len(list.Data) {
		stop = len(list.Data) - 1
	}
	if start > stop {
		return [][]byte{}, nil
	}

	return list.Data[start : stop+1], nil
}

func (s *Store) HSet(key, field string, value []byte) (int, error) {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()
	if v, ok := shard.data[key]; ok {
		if v.Value.Type() != hashType {
			return 0, ErrWrongType
		}

		_, exists := shard.data[key].Value.(HashValue).Data[field]
		shard.data[key].Value.(HashValue).Data[field] = value
		if exists {
			return 0, nil
		}

		return 1, nil
	}

	shard.data[key] = &Entry{
		Value: HashValue{Data: map[string][]byte{
			field: value,
		}},
	}

	return 1, nil
}

func (s *Store) HGet(key, field string) ([]byte, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return nil, nil
	}

	if shard.data[key].Value.Type() != hashType {
		return nil, ErrWrongType
	}

	value, exists := shard.data[key].Value.(HashValue).Data[field]
	if !exists {
		return nil, nil
	}

	return value, nil
}

func (s *Store) HDel(key string, fields ...string) (int, error) {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, ok := shard.data[key]; !ok {
		return 0, nil
	}

	if shard.data[key].Value.Type() != hashType {
		return 0, ErrWrongType
	}

	count := 0
	for _, field := range fields {
		_, exists := shard.data[key].Value.(HashValue).Data[field]
		if exists {
			delete(shard.data[key].Value.(HashValue).Data, field)
			count++
		}
	}

	if len(shard.data[key].Value.(HashValue).Data) == 0 {
		delete(shard.data, key)
	}

	return count, nil
}

func (s *Store) HGetAll(key string) (map[string][]byte, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return make(map[string][]byte), nil
	}

	if shard.data[key].Value.Type() != hashType {
		return nil, ErrWrongType
	}

	return shard.data[key].Value.(HashValue).Data, nil
}

func (s *Store) HKeys(key string) ([]string, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return []string{}, nil
	}

	if shard.data[key].Value.Type() != hashType {
		return nil, ErrWrongType
	}

	keys := make([]string, 0, len(shard.data[key].Value.(HashValue).Data))

	for k := range shard.data[key].Value.(HashValue).Data {
		keys = append(keys, k)
	}

	return keys, nil
}

func (s *Store) HVals(key string) ([][]byte, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return [][]byte{}, nil
	}

	if shard.data[key].Value.Type() != hashType {
		return nil, ErrWrongType
	}

	values := make([][]byte, 0, len(shard.data[key].Value.(HashValue).Data))

	for _, v := range shard.data[key].Value.(HashValue).Data {
		values = append(values, v)
	}

	return values, nil
}

func (s *Store) HLen(key string) (int, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return 0, nil
	}

	if shard.data[key].Value.Type() != hashType {
		return 0, ErrWrongType
	}

	return len(shard.data[key].Value.(HashValue).Data), nil
}

func (s *Store) HExists(key, field string) (bool, error) {
	shard := s.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if _, ok := shard.data[key]; !ok {
		return false, nil
	}

	if shard.data[key].Value.Type() != hashType {
		return false, ErrWrongType
	}

	if _, exists := shard.data[key].Value.(HashValue).Data[field]; !exists {
		return false, nil
	}

	return true, nil
}

func (s *Store) StartExpirationLoop(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := range s.shards {
				go expirationLoop(&s.shards[i])
			}
		}
	}
}

func expirationLoop(shard *Shard) {
	for range 5 {
		itterations := 0
		var checked, expired int

		shard.mu.RLock()
		for key, value := range shard.data {
			if itterations == 20 {
				break
			}

			if value.ExpireAt <= time.Now().UnixNano() && value.ExpireAt != 0 {
				expired++
				shard.mu.RUnlock()
				shard.mu.Lock()
				entry := shard.data[key]
				if entry != nil && entry.ExpireAt != 0 && entry.ExpireAt <= time.Now().UnixNano() {
					delete(shard.data, key)
				}
				shard.mu.Unlock()
				shard.mu.RLock()
			}

			itterations++
			checked++
		}

		shard.mu.RUnlock()

		if expired < checked/4 {
			return
		}
	}
}
