package main

import "sync"

type call struct {
	wg     sync.WaitGroup
	result URLResult
}

type StampedePreventer struct {
	mu       sync.Mutex
	inFlight map[string]*call
}

func NewStampedePreventer() *StampedePreventer {
	return &StampedePreventer{
		inFlight: make(map[string]*call),
	}
}

func (sp *StampedePreventer) Fetch(url string, fetchFunc func(string) URLResult) URLResult {
	sp.mu.Lock()

	if c, found := sp.inFlight[url]; found {
		sp.mu.Unlock()
		c.wg.Wait()
		return c.result
	}

	c := &call{}
	c.wg.Add(1)
	sp.inFlight[url] = c
	sp.mu.Unlock()

	c.result = fetchFunc(url)
	c.wg.Done()

	sp.mu.Lock()
	delete(sp.inFlight, url)
	sp.mu.Unlock()

	return c.result
}
