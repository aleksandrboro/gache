package datastruct

import "math/rand/v2"

const (
	maxLevel = 32
)

type SkipList struct {
	head   *SkipListNode
	level  int
	lenght int
}

type SkipListNode struct {
	score   float64
	member  string
	forward []*SkipListNode
}

func NewSkipList() *SkipList {
	return &SkipList{
		head: &SkipListNode{},
	}
}

func RandomLevel() int {
	level := 1

	for float32(rand.Int32N(2)) < 0.5 {
		if level == maxLevel {
			return level
		}

		level++
	}

	return level
}

func (sl *SkipList) Insert(score float64, member string) {
	
}
