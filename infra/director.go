package infra

import (
  "gossip/sipmsg"
  "sync"
  "regexp"
  "log"
  "time"
  "runtime"
)

// used as index for the different maps
const (
	Number = iota
	CallId
	Via
	DirectoryEnd
)

var (
	// the maps from tags to strings
	DirectorChans []director
	// the related mutex for access serialization
	NumberLock []sync.RWMutex
	viaReg     *regexp.Regexp
	siptelReg  *regexp.Regexp
)

// maps a certain tag to a channel for gossip items
type director map[string]chan *sipmsg.Item

// init gets called during initialization, each init function in each file
func init() {

	NumberLock = make([]sync.RWMutex, DirectoryEnd)
	DirectorChans = make([]director, DirectoryEnd)
	for i := 0; i < DirectoryEnd; i++ {
		DirectorChans[i] = make(director)
	}
	// starting a background goroutine
	go cleanUpDirector()
	// compiling some regular expression
	re, err := regexp.Compile("branch=([^; ]*)")
	if err != nil {
		log.Fatal(err)
	}
	viaReg = re
	re, err = regexp.Compile(":([^@; ]*)")
	if err != nil {
		log.Fatal(err)
	}
	siptelReg = re
}


// cleanUpDirector removes idle channels from the maps,
// idle means that the channel is full,
// when the goroutines serving the channels end, they might have been registered several times
func cleanUpDirector() {
	for {
		// until end of time
		for i := 0; i < DirectoryEnd; i++ {
			time.Sleep(time.Second)
			// the keys to remove
			k2d := []string{}
			NumberLock[i].RLock()
			for k, v := range DirectorChans[i] {
				select {
				case v <- nil:
					// everything ok
				default:
					// channel is filled == no reader left
					k2d = append(k2d, k)
				}
			}
			NumberLock[i].RUnlock()
			NumberLock[i].Lock()
			for _, k := range k2d {
				delete(DirectorChans[i], k)
			}
			NumberLock[i].Unlock()
			runtime.Gosched()
		}
	}
}

// SendItem sends the item <it> on the channel indicated by <dir> and <key>
// it needs to obtain a reading lock on the map,
// since maps are not thread safe
func SendItem(dir int, key string, it *sipmsg.Item) (ok bool) {
	NumberLock[dir].RLock()
	ch := DirectorChans[dir][key]
	NumberLock[dir].RUnlock()
	if ch == nil {
		return false
	}
	select {
	// trying the channel, it full, we return false
	case ch <- it:
		ok = true
	default:
		ok = false
	}
	return
}

