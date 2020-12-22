package utils

import (
	"sync"
	"time"
  "math"
)

var (
	// ticker sends time msg in certain intervals
	ticker = time.NewTicker(time.Second)
	// number of concurrent calls - compared with cfg.Concurrent
	current int32
	// Lock and Signal
	cond *sync.Cond = sync.NewCond(new(sync.Mutex))
  // from configuration
  maxConcurrent int32=math.MaxInt32
  // 
  wg sync.WaitGroup
)

// Limiter configures the rate and max concurrent number of running tests
func Limiter(cfg *Config){
  if cfg.Rate > 0 {
  di := int64(time.Second) / int64(cfg.Rate)
  d := time.Duration(di)
  ticker = time.NewTicker(d)
  }
  if cfg.Concurrent > 0 {
    maxConcurrent=cfg.Concurrent
  }
}

// Claim claims a spot and a tick from the rate
func Claim(){
  cond.L.Lock()
	// no one can change current right now
	for current >= maxConcurrent {
		cond.Wait() // wait unlocks, waits and locks
	}
	current++
	<-ticker.C
  wg.Add(1)
	cond.L.Unlock()
}

// Release frees a claim and decrements the number of concurrent runners
func Release(){
	cond.L.Lock()
	current--
	cond.Broadcast() // FetchLimited's wait can continue
	cond.L.Unlock()
  wg.Done()
}

// Wait waits for all tests to be done
func Wait() {
  wg.Wait()
}
