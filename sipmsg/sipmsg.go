package sipmsg

import (
  "regexp"
  "strconv"
  "log"
  "strings"
  "errors"
)

const (
  ReqUnknown=iota
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
  // SipType: either from enum or 3digit
  SipType int
  Headers MsgHeaders
  Direction int // enum 
}

func (msg *SipMsg) Retrieve(str string) error{
  if msg.Headers == nil {
    msg.Headers= make(MsgHeaders)
  }
  str=strings.ReplaceAll(str,"\r\n","\n")
  strs:=strings.Split(str,"\n")
  for _,str=range strs {
    str=strings.TrimSpace(str)
    log.Printf("str '%s' is %d long\n", str, len(str))
    if len(str)==0 {
      continue
    }
    parts:=strings.SplitN(str,":",2)
    if len(parts)<2 {
      return errors.New(str)
    }
    key:=strings.TrimSpace(parts[0])
    value:=strings.TrimSpace(parts[1])
    msg.Headers[key]=append(msg.Headers[key],value)
  }
  return nil
}
