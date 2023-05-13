package main

import (
	"container/heap"
)

type PodPriority struct {
	id       string
	priority int
	index    int
}

type PriorityQ []*PodPriority

func (pq PriorityQ) Len() int {
	return len(pq)
}

// Returns true for lowest priority.
func (pq PriorityQ) Less(a, b int) bool {
	return pq[a].priority < pq[b].priority
}

func (pq PriorityQ) Swap(a, b int) {
	if a < 0 || a >= len(pq) || b < 0 || b >= len(pq) {
		return
	}

	pq[a], pq[b] = pq[b], pq[a]
	pq[a].index, pq[b].index = a, b
}

func (pq *PriorityQ) Push(a interface{}) {
	pod := a.(*PodPriority)
	pod.index = len(*pq)
	*pq = append(*pq, pod)
}

func (pq *PriorityQ) Pop() interface{} {
	epq := *pq
	l := len(epq)
	if l < 1 {
		return nil
	}
	lastPod := epq[l-1]
	epq[l-1] = nil
	lastPod.index = -1
	*pq = epq[:l-1]
	return lastPod
}

func (pq *PriorityQ) update(pod PodPriority, id string, priority int) {
	pod.id = id
	pod.priority = priority
	heap.Fix(pq, pod.index)
}
