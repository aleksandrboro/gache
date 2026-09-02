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
	Score   float64
	Member  string
	Forward []*SkipListNode
}

func NewSkipList() *SkipList {
	return &SkipList{
		head: &SkipListNode{
			Forward: make([]*SkipListNode, maxLevel),
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
		for current.Forward[level] != nil && current.Forward[level].Score < score {
			current = current.Forward[level]
		}

		update[level] = current
		level--
	}

	newNode := &SkipListNode{
		Score:   score,
		Member:  member,
		Forward: make([]*SkipListNode, nodeLevel),
	}

	for i := range nodeLevel {
		newNode.Forward[i] = update[i].Forward[i]
		update[i].Forward[i] = newNode
	}

	sl.length++
}

func (sl *SkipList) Delete(score float64, member string) {
	founded := false
	level := sl.level - 1
	current := sl.head

	for level >= 0 {
		for current.Forward[level] != nil && current.Forward[level].Score < score {
			current = current.Forward[level]
		}

		if current.Forward[level] != nil && current.Forward[level].Member == member && current.Forward[level].Score == score {
			current.Forward[level] = current.Forward[level].Forward[level]
			founded = true
		}

		level--
	}

	if founded {
		sl.length--
	}
}

func (sl *SkipList) GetByRank(start, stop int) []SkipListNode {
	if start > stop {
		return []SkipListNode{}
	}

	if stop == -1 {
		stop = sl.length - 1
	}

	position := 0
	current := sl.head.Forward[0]
	result := make([]SkipListNode, 0, stop-start+1)

	for position < sl.length {
		if position > stop {
			break
		}

		if position >= start && position <= stop {
			result = append(result, *current)
		}

		position++
		current = current.Forward[0]
	}

	return result
}
