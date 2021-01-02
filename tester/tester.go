package tester

import (
	"gossip/infra"
	"gossip/sipmsg"
	"gossip/utils"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	// testLocks is a map to Mutex references
	testLocks = make(map[*utils.SingleTest]*sync.Mutex)
	// Running: if false then stop all tests
	Running bool
)

// Tester executes a single test, coordinating goroutines that execute the parties
type Tester struct {
	test *utils.SingleTest
	// the selected remote side
	Remote      string
	RemoteParts []string
	// the corresponding local site
	Local      string
	LocalParts []string
	// points to an entry in testLocks, shared as reference
	lock *sync.Mutex
	// indicates that the test is running - good for stopping goroutines
	running bool
	// implementing barriers - simply doing Done()+Wait()
	// barrier for setup section
	wg_setup sync.WaitGroup
	// barrier for the test execution action
	wg_run sync.WaitGroup
	// barrier for the cleanup - this one indicates all goroutines are done
	wg_down sync.WaitGroup
}

// Run executes the master part of the test
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
	// wait for all goroutines to finish
	te.wg_down.Wait()
}

// CreatePartyTest creates each party's own goroutine with data
func (te *Tester) CreatePartyTest(cp int) (pt *PartyTest) {
	pt = new(PartyTest)
	pt.te = te
	pt.party = te.test.CallParties[cp]
	pt.builder = newBuilder()
	pt.msgsMap = make(map[string]*sipmsg.SipMsg)
	return
}

// PartyTest is test datastructure for a call party
type PartyTest struct {
	msgs     []*sipmsg.SipMsg
	msgsMap  map[string]*sipmsg.SipMsg
	next     string
	party    *utils.CallParty
	previous string
	si       int
	steps    map[string]int
	te       *Tester
	builder  *Builder
}

// RunCall is the goroutine for one call party, almost deterministic inside this one
func (pt *PartyTest) RunCall() {
	number := pt.party.Number
	log.Println("setting up for " + number)
	pt.steps = make(map[string]int)
	for i, ci := range pt.party.Steps {
		if len(ci.Alias) > 0 {
			pt.steps[ci.Alias] = i
		}
	}
	// TODO register party's number in director
	// barrier
	pt.te.wg_setup.Done()
	pt.te.wg_setup.Wait()
	// the run
	for pt.si = 0; pt.si < len(pt.party.Steps); pt.advance() {
		if !pt.te.running {
			break
		}
		pt.execute(pt.party.Steps[pt.si])
		// TODO implement optional messages, like a Cancel or a Bye
	}
	// first party that finishes finishes the whole test
	if pt.te.running {
		// give the others time to end
		time.Sleep(100 * time.Millisecond)
		pt.te.running = false
	}
	// barrier
	pt.te.wg_run.Done()
	pt.te.wg_run.Wait()
	// cleanup
	// and signal end
	pt.te.wg_down.Done()
}

// advance finds out where to go next to
func (pt *PartyTest) advance() {
	if len(pt.next) > 0 {
		pt.si = pt.steps[pt.next]
		pt.next = ""
	} else {
		pt.si++
	}
}

// logError logs a fatal error and also gives and indication where in which test it failed
func (pt *PartyTest) logError(step *utils.CallStep, err error) {
	party := pt.party
	number := party.Number
	test := party.RLsingleTest
	suite := test.RLtestSuite
	log.Fatalf("%s/%s:%s %s", suite.Name, test.Name, number, err)
}

// execute carries out one step, based on what is specified
func (pt *PartyTest) execute(step *utils.CallStep) {
	// #########################################################
	// ##### looking for a delay
	if len(step.Delay) > 0 {
		dur, err := time.ParseDuration(step.Delay)
		if err != nil {
			pt.logError(step, err)
		}
		ch := time.After(dur)
		for pt.te.running {
			select {
			// timed out
			case <-ch:
				break
				// TODO when message arrives, it is a failure

			}
		}
	}
	// #########################################################
	// ##### looking for an out message
	if len(step.Out) > 0 {
		var item *sipmsg.Item
		req := sipmsg.SipType(step.Out)
		if req < 100 {
			item = pt.makeRequest(step, req)
		} else {
			log.Fatalln("### not yet implemented")
		}
		// TODO send the message to the provider to send it further
		infra.Transmit(item)
	}
	// #########################################################
}

// makeRequest creates a SIP message
func (pt *PartyTest) makeRequest(step *utils.CallStep, req int) *sipmsg.Item {
	var prev *sipmsg.SipMsg
	if len(step.Previous) > 0 {
		prev = pt.msgsMap[step.Previous]
	} else {
		l := len(pt.msgs) - 1
		switch {
		case l == -1:
			prev = nil
		case req >= 100:
			// TODO search for a non finished transaction
			log.Fatal("### not yet implemented")
		default:
			prev = pt.msgs[l]
		}

	}
	b := builder(prev, step, req, pt.te)
	if len(step.SdpTags) > 0 {
		b.createSDP(step)
	}
	item := b.getItem()
	item.RemoteEP = pt.te.Remote
	return item
}

// Create a new Tester
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
	// finding a remote
	te.Local, te.Remote = cfg.GetTransport()
	te.LocalParts = strings.Split(te.Local, "/")
	te.RemoteParts = strings.Split(te.Remote, "/")
	//
	utils.Claim()
	return
}
