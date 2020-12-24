package tester

import(
  "gossip/sipmsg"
  "gossip/utils"
  "text/template"
  "log"
  "bytes"
  "strings"
  "os"
)

var (
  funcMap template.FuncMap
)

func init(){
  funcMap=make(template.FuncMap)
  funcMap["RandStr"]=utils.RandString
  
}

type Builder struct {
  templ *template.Template
  msg *sipmsg.SipMsg
}

type BData struct {
  CallId string
}

func build(prev *sipmsg.SipMsg,ci *utils.CallItem, req int) (b *Builder){
  b=new(Builder)
  b.msg=new(sipmsg.SipMsg)
  b.msg.Prev=prev
  data:=new(BData)
  var err error
  for p:=prev;p!=nil;p=p.Prev {
    if p.Prev == nil {
      b.msg.Req=p
      break
    }
  }
  if prev != nil {
    data.CallId=prev.Transaction.Call.CallId
  } else {
    
  }
  templ:=template.New("SipMsg")
  templ= templ.Funcs(funcMap)
  cp:=ci.RLcallParty
  st:=cp.RLsingleTest
  ts:=st.RLtestSuite
  fn:=ts.RLfileName
  hdrs:=make(sipmsg.MsgHeaders)
  if len(ci.Headers) > 0{
    templ,err=templ.Parse(ci.Headers)
    if err != nil {
      log.Printf("wrong template in %s: %s/%s/%s template below\n%s\n",fn,ts.Name,st.Name,cp.Number,ci.Headers)
      log.Fatal(err)
    }
    bb:=new(bytes.Buffer)
    err=templ.Execute(bb,data)    
    if err != nil {
      log.Printf("error in %s: %s/%s/%s template below\n%s\n",fn,ts.Name,st.Name,cp.Number,ci.Headers)
      log.Fatal(err)
    }
    str:=bb.String()
    str=strings.ReplaceAll(str,"\r\n","\n")
    strs:=strings.Split(str,"\n")
    for _,str=range strs {
      str=strings.TrimSpace(str)
      if len(str)==0 {
        continue
      }
      parts:=strings.SplitN(str,":",2)
      if len(parts)<2 {
        log.Printf("the template isn't right %s: %s/%s/%s\n%s",fn,ts.Name,st.Name,cp.Number,str)
        os.Exit(2)
      }
      key:=strings.TrimSpace(parts[0])
      value:=strings.TrimSpace(parts[1])
      hdrs[key]=append(hdrs[key],value)
    }
  }
  b.templ=templ
  return
}
