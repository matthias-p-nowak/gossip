package tester

import (
  "gossip/utils"
  "sync"
  "log"
)

var (
  testLocks=make(map[*utils.SingleTest] *sync.Mutex)
)

type Tester struct {
  test *utils.SingleTest
  lock *sync.Mutex
  running bool
  wg_setup sync.WaitGroup
  wg_run sync.WaitGroup
  wg_down sync.WaitGroup
}

func (te *Tester) Run() {
  defer utils.Release()
  defer te.lock.Unlock()
  cps:=len(te.test.CallParties)
  te.wg_setup.Add(cps)
  te.wg_run.Add(cps)
  te.wg_down.Add(cps)
  for cp:=0;cp<cps; cp++ {
    go te.RunCall(cp)
  }
  te.wg_down.Wait()
}

func (te *Tester) RunCall(cp int) {
  // prep
  party:=te.test.CallParties[cp]
  number:=party.Number
  log.Println("setting up for "+number)
  steps:=make(map[string]int)
  for i,ci:=range party.Steps {
    if len(ci.Alias)>0 {
      steps[ci.Alias]=i
    }
  }
  // barrier
  te.wg_setup.Done()
  te.wg_setup.Wait()
  // the run
  
  // barrier
  te.wg_run.Done()
  te.wg_run.Wait()
  // cleanup
  // and signal end
  te.wg_down.Done()
}


func Create(test *utils.SingleTest) (te * Tester){
  // Create is called in main thread
  lock, ok := testLocks[test]
  if !ok {
    lock=new(sync.Mutex)
    testLocks[test]=lock
  }
  lock.Lock()
  te=new(Tester)
  te.test=test
  te.lock=lock // avoiding to lookup the map
  utils.Claim()
  return
}
