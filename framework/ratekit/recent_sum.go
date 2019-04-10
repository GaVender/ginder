package ratekit

/*
计算最近收集的数字的和
*/

import (
	"sync"
)

type RecentSum struct {
	counts 	[]int
	len  	int
	mu   	*sync.RWMutex
}

func NewRecentSum(len int) *RecentSum {
	if len <= 0 {
		len = 5
	}
	
	r := &RecentSum{}
	r.len = len
	r.mu = &sync.RWMutex{}
	r.counts = make([]int, len, len)
	
	for i := 0; i < len; i++ {
		r.counts[i] = 0
	}
	
	return r
}

func (r *RecentSum) Put(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.counts = append(r.counts[1:r.len], n)
}

func (r *RecentSum) Sum() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	sum := 0
	
	for _, e := range r.counts {
		sum += e
	}
	
	return sum
}