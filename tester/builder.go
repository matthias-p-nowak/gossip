package tester

import(
  "gossip/sipmsg"
  "gossip/utils"
  "text/template"
  "log"
  "bytes"
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
  Msg *sipmsg.SipMsg
  CallId string
  LocalTag string
  RemoteTag string
}

func (d *BData) fill(msg *sipmsg.SipMsg){
  d.Msg=msg
  d.CallId=msg.Transaction.Call.CallId
  dir:=msg.Direction
  st:=msg.SipType
  if (dir== sipmsg.DirIn) == (st < 100){
    log.Println("from == remote")
  } else {
    log.Println(" to == remote")
  }
  // 
}

func (d *BData) fillDefault(){
  d.CallId=utils.RandString(10)
  d.LocalTag=utils.RandString(10)
}

func build(prev *sipmsg.SipMsg,ci *utils.CallItem, req int) (b *Builder){
  b=new(Builder)
  b.msg=new(sipmsg.SipMsg)
  b.msg.Prev=prev
  b.msg.SipType=req
  data:=new(BData)
  var err error
  if req >= 100 {
    for p:=prev;p!=nil;p=p.Prev {
      if p.Prev == nil || p.SipType<100 {
        b.msg.Req=p
        break
      }
    }
  }
  if prev != nil {
    data.fill(prev)
  } else {
    data.fillDefault()
  }
  templ:=template.New("SipMsg")
  templ= templ.Funcs(funcMap)
  cp:=ci.RLcallParty
  st:=cp.RLsingleTest
  ts:=st.RLtestSuite
  fn:=ts.RLfileName
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
    err=b.msg.Retrieve(str)
    if err != nil {
      log.Printf("error in %s: %s/%s/%s expanded template below\n%s\n",fn,ts.Name,st.Name,cp.Number,str)
      log.Fatal(err)
    }
  }
  b.templ=templ
  return
}
