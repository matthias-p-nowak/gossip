package sipmsg

import (
  "regexp"
  "strconv"
  "log"
  "strings"
)

const (
  ReqUnknown=iota
  ReqInvite
  ReqCancel
  ReqBye
  ReqAck
  ReqPrack
)

var (
 ThreeDigits *regexp.Regexp
)

func init(){
  re,err := regexp.Compile("[0-9]{3}")
  if err!=nil { log.Fatal(err)}
  ThreeDigits=re
}

func SipType(s string) int {
  if ThreeDigits.MatchString(s){
    i,err := strconv.Atoi(s)
    if err != nil {
      log.Fatal(err)
    }
    return i
  }
  s=strings.ToLower(s)
  switch(s){
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



type SipCall struct {
  CallId string
  CallSeq int
}

type SipTransaction struct {
  Call *SipCall
}


// MsgHeaders: Headers might have multiple values (via, record-routes, route),
// hence it contains a list of values
type MsgHeaders map[string][]string


type SipMsg struct {
  Transaction *SipTransaction
  Prev *SipMsg
  Req *SipMsg
}
