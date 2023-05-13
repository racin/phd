package main

type Ratelimiter struct {
	limit    int
	resource chan struct{}
}

func NewRatelimiter(limit int) *Ratelimiter {
	return &Ratelimiter{limit: limit, resource: make(chan struct{}, limit)}
}

func (rl *Ratelimiter) GetLimit() int {
	return rl.limit
}

func (rl *Ratelimiter) Acquire() {
	rl.resource <- struct{}{}
}

func (rl *Ratelimiter) Release() {
	<-rl.resource
}
