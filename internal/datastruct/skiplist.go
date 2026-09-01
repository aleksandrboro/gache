package datastruct

import "math/rand/v2"

const (
	maxLevel = 32
)

type SkipList struct {
	head   *SkipListNode
	level  int
	length int
}

type SkipListNode struct {
	score   float64
	member  string
	forward []*SkipListNode
}

func NewSkipList() *SkipList {
	return &SkipList{
		head: &SkipListNode{
			forward: make([]*SkipListNode, maxLevel),
		},
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
	update := make([]*SkipListNode, maxLevel)

	nodeLevel := RandomLevel()
	if nodeLevel > sl.level {
		sl.level = nodeLevel
	}

	level := sl.level - 1
	current := sl.head

	for level >= 0 {
		for current.forward[level] != nil && current.forward[level].score < score {
			current = current.forward[level]
		}

		update[level] = current
		level--
	}

	newNode := &SkipListNode{
		score:   score,
		member:  member,
		forward: make([]*SkipListNode, nodeLevel),
	}

	for i := range nodeLevel {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	sl.length++
}
