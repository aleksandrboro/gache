package datastruct

type ZSet struct {
	sl  *SkipList
	set map[string]float64
}

func NewZSet() *ZSet {
	return &ZSet{
		sl:  NewSkipList(),
		set: make(map[string]float64),
	}
}

func (zs *ZSet) Add(score float64, member string) int {
	if oldScore, ok := zs.set[member]; ok {
		zs.sl.Delete(oldScore, member)
		zs.sl.Insert(score, member)
		zs.set[member] = score
		return 0
	} else {
		zs.sl.Insert(score, member)
		zs.set[member] = score
		return 1
	}
}

func (zs *ZSet) Rem(member string) bool {
	if score, ok := zs.set[member]; ok {
		zs.sl.Delete(score, member)
		delete(zs.set, member)
		return true
	}

	return false
}

func (zs *ZSet) Score(member string) (float64, bool) {
	score, ok := zs.set[member]

	return score, ok
}

func (zs *ZSet) Len() int {
	return zs.sl.length
}

func (zs *ZSet) Range(start, stop int) []SkipListNode {
	return zs.sl.GetByRank(start, stop)
}
