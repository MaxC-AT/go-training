package localcache

import (
	"container/heap"
)

// priorityQueue implements heap.Interface and holds cacheItems.
type priorityQueue []*cacheItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].exp < pq[j].exp
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].idx = i
	pq[j].idx = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*cacheItem)
	item.idx = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.idx = -1
	*pq = old[0 : n-1]
	return item
}

// reset updates the priority queue when the item is updated to ensure invariant.
func (pq *priorityQueue) reset(item *cacheItem, value interface{}, exp int64) {
	item.exp = exp
	item.val = value
	heap.Fix(pq, item.idx)
}
