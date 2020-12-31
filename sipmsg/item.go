package sipmsg

import (
	"log"
	"strconv"
	"strings"
	"time"
)

// Retransmission count
const (
	NoRetrans = iota
	ReTrOnce
	ReTrFirst
	ReTrSecond
	ReTrThird
	ReTrFourth
	ReTrFifth
	ReTrSixth
	ReTrSeventh
	ReTrEnd
)

// Item combines the message with related information
type Item struct {
	Msg      *SipMsg
	LocalEP  string
	RemoteEP string
	// channel that answers should be send over
	Ch chan *Item
	// the raw packet send over IP
	RawMsg []byte
	// hash of the raw packet - for identifying retransmissions
	Hash uint32
	// RetrCount: retransmission count
	RetrCount int
}

// delaySend for retransmissions, the same message must be send again after a certain interval
func delaySend(dur time.Duration, ch chan *Item, gi *Item) {
	time.AfterFunc(dur, func() {
		select {
		case ch <- gi:
		default:
			// channel is full - most likely abandoned
		}
	})
}

func (item *Item) Prepare4Send() {
	if len(item.RawMsg) > 0 {
		return
	}
	body := strings.Join(item.Msg.BodyList, "\r\n") + "\r\n"
	cl := len(body)
	headers := item.Msg.Headers
	headers["Content-Length"] = []string{strconv.Itoa(cl)}
	var strs []string
	strs = append(strs, item.Msg.StartLine)
	for k, l := range headers {
		for _, v := range l {
			strs = append(strs, k+": "+v)
		}
	}
	str := strings.Join(strs, "\r\n") + "\r\n\r\n" + body
	log.Printf("expanded string is \n---- begin -----\n%s----- end ----\n", str)

}
