package storage

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetGet(t *testing.T) {
	store := NewStore()

	store.Set("key", []byte("value"))
	v, ok := store.Get("key")
	require.Equal(t, []byte("value"), v)
	require.True(t, ok)

	v, ok = store.Get("k")
	require.Empty(t, v)
	require.False(t, ok)
}

func TestDel(t *testing.T) {
	store := NewStore()

	store.Set("key1", []byte("value1"))
	store.Set("key2", []byte("value2"))
	store.Set("key3", []byte("value3"))

	count := store.Del("key1")
	require.Equal(t, 1, count)

	count = store.Del("key2", "key3")
	require.Equal(t, 2, count)

	count = store.Del("key4")
	require.Equal(t, 0, count)
}

func TestExists(t *testing.T) {
	store := NewStore()

	store.Set("key1", []byte("value1"))
	store.Set("key2", []byte("value2"))
	store.Set("key3", []byte("value3"))

	count := store.Exists("key1")
	require.Equal(t, 1, count)

	count = store.Exists("key1", "key2", "key3")
	require.Equal(t, 3, count)

	count = store.Exists("key4")
	require.Equal(t, 0, count)
}

func TestConcurrentSetGet(t *testing.T) {
	store := NewStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			value := []byte(fmt.Sprintf("value-%d", id))

			store.Set(key, value)
			val, ok := store.Get(key)
			require.True(t, ok)
			require.Equal(t, value, val)
		}(i)
	}

	wg.Wait()
}

func TestExistsWithDuplicates(t *testing.T) {
	store := NewStore()
	store.Set("a", []byte("1"))
	store.Set("b", []byte("2"))

	count := store.Exists("a", "b", "a")
	require.Equal(t, 3, count)
}

func TestSetOverwrite(t *testing.T) {
	store := NewStore()

	store.Set("key", []byte("old"))
	store.Set("key", []byte("new"))

	val, ok := store.Get("key")
	require.True(t, ok)
	require.Equal(t, []byte("new"), val)
}

func TestEmptyStrings(t *testing.T) {
	store := NewStore()

	store.Set("", []byte("value"))
	val, ok := store.Get("")
	require.True(t, ok)
	require.Equal(t, []byte("value"), val)

	store.Set("key", []byte(""))
	val, ok = store.Get("key")
	require.True(t, ok)
	require.Empty(t, val)
}
