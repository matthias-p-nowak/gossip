package tester

import (
	"gossip/sipmsg"
	"gossip/utils"
	"log"
	"sync"
)

var (
	testLocks = make(map[*utils.SingleTest]*sync.Mutex)
	remIdx    int
)

type Tester struct {
	test     *utils.SingleTest
	Remote   string // the selected remote side
	lock     *sync.Mutex
	running  bool
	wg_setup sync.WaitGroup
	wg_run   sync.WaitGroup
	wg_down  sync.WaitGroup
}

func (te *Tester) Run() {
	defer utils.Release()
	defer te.lock.Unlock()
	te.running = true
	cps := len(te.test.CallParties)
	te.wg_setup.Add(cps)
	te.wg_run.Add(cps)
	te.wg_down.Add(cps)
	for cp := 0; cp < cps; cp++ {
		pt := te.CreatePartyTest(cp)
		go pt.RunCall()
	}
	te.wg_down.Wait()
}

func (te *Tester) CreatePartyTest(cp int) (pt *PartyTest) {
	pt = new(PartyTest)
	pt.te = te
	pt.party = te.test.CallParties[cp]
	return
}

type PartyTest struct {
	te *Tester

	party    *utils.CallParty
	next     string
	previous string
	si       int
	steps    map[string]int
	msgs     []*sipmsg.SipMsg
}

func (pt *PartyTest) RunCall() {
	number := pt.party.Number
	log.Println("setting up for " + number)
	pt.steps = make(map[string]int)
	for i, ci := range pt.party.Steps {
		if len(ci.Alias) > 0 {
			pt.steps[ci.Alias] = i
		}
	}
	// TODO get a remote side

	// barrier
	pt.te.wg_setup.Done()
	pt.te.wg_setup.Wait()
	// the run
	for pt.si = 0; pt.si < len(pt.party.Steps); pt.advance() {
		if !pt.te.running {
			break
		}
		pt.execute(pt.party.Steps[pt.si])
	}
	// barrier
	pt.te.wg_run.Done()
	pt.te.wg_run.Wait()
	// cleanup
	// and signal end
	pt.te.wg_down.Done()
}

func (pt *PartyTest) advance() {
	if len(pt.next) > 0 {
		pt.si = pt.steps[pt.next]
	} else {
		pt.si++
	}
}

func (pt *PartyTest) execute(ci *utils.CallItem) {
	if len(ci.Out) > 0 {
    var msg *sipmsg.SipMsg
		req := sipmsg.SipType(ci.Out)
		if req < 100 {
			msg = pt.makeRequest(ci, req)
		}
    it:=sipmsg.CreateItem(msg)
	// TODO send the message to the provider to send it further
  
	}
}

func (pt *PartyTest) makeRequest(ci *utils.CallItem, req int) (msg *sipmsg.SipMsg) {
	msg = new(sipmsg.SipMsg)
	var prev *sipmsg.SipMsg
	// TODO if previous is specified, use that one
	// else
	{
		l := len(pt.msgs) 
    if l >= 2 {
			prev = pt.msgs[l-2]
		}
	}
	b := buildSip(prev, ci, req, pt.te.Remote)
	if len(ci.SdpTags) > 0 {
		b.createSDP(ci)
	}
	return
}

func Create(test *utils.SingleTest, cfg *utils.Config) (te *Tester) {
	// Create is called in main thread
	lock, ok := testLocks[test]
	if !ok {
		lock = new(sync.Mutex)
		testLocks[test] = lock
	}
	lock.Lock()
	te = new(Tester)
	te.test = test
	te.lock = lock // avoiding to lookup the map
	remotes := cfg.Remote
	remIdx = (remIdx + 1) % len(remotes)
	te.Remote = remotes[remIdx]
	utils.Claim()
	return
}
