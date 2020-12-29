package sipmsg

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

const (
	ReqUnknown = iota
	ReqInvite
	ReqCancel
	ReqBye
	ReqAck
	ReqPrack
)

const (
	DirIn = iota
	DirOut
)

var (
	// ThreeDigits matches 3 digits
	ThreeDigits *regexp.Regexp
)

func init() {
	re, err := regexp.Compile("[0-9]{3}")
	if err != nil {
		log.Fatal(err)
	}
	ThreeDigits = re
}

// SipType turns the string into an enum
func SipType(s string) int {
	if ThreeDigits.MatchString(s) {
		i, err := strconv.Atoi(s)
		if err != nil {
			log.Fatal(err)
		}
		return i
	}
	s = strings.ToLower(s)
	switch s {
	case "invite":
		return ReqInvite
	case "cancel":
		return ReqCancel
	case "bye":
		return ReqBye
	case "ack":
		return ReqAck
	case "prack":
		return ReqPrack
	}
	return ReqUnknown
}

// SipT2S turns the enum into a string
func SipT2S(req int) string {
	switch req {
	case ReqInvite:
		return "INVITE"
	default:
		return fmt.Sprintf("%d", req)
	}
}

// SipCall Call related info
type SipCall struct {
	CallId  string
	CallSeq int
}

// SipTransaction the transaction
type SipTransaction struct {
	Call      *SipCall
	LocalTag  string
	Local     string
	RemoteTag string
	Remote    string
}

// MsgHeaders: Headers might have multiple values (via, record-routes, route),
// hence it contains a list of values
type MsgHeaders map[string][]string

// SipMsg is the SIP message
type SipMsg struct {
	Transaction *SipTransaction
	Prev        *SipMsg
	Req         *SipMsg
	// SipType: either from enum or 3digit
	SipType int
	// StartLine is either the request line or the reponse line
	StartLine string
	// Headers maps string to list of header values
	Headers MsgHeaders
	// BodyList is the list of lines
	BodyList []string
	// in or out
	Direction int // enum
}

func (msg *SipMsg) Retrieve(str string) error {
	if msg.Headers == nil {
		msg.Headers = make(MsgHeaders)
	}
	str = strings.ReplaceAll(str, "\r\n", "\n")
	strs := strings.Split(str, "\n")
	for i, str := range strs {
		str = strings.TrimSpace(str)
		// log.Printf("str '%s' is %d long\n", str, len(str))
		if len(str) == 0 {
			msg.BodyList = strs[i+1:]
			break
		}
		parts := strings.SplitN(str, ":", 2)
		if len(parts) < 2 {
			return errors.New(str)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		msg.Headers[key] = append(msg.Headers[key], value)
	}
	return nil
}
