package tester

import (
	"bytes"
	"fmt"
	"gossip/sipmsg"
	"gossip/utils"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

var (
	funcMap template.FuncMap
)

func init() {
	funcMap = make(template.FuncMap)
	funcMap["RandStr"] = utils.RandString
	// TODO add more useful functions here
}

type Builder struct {
	templ *template.Template
	msg   *sipmsg.SipMsg
	data  *BData
}

type BData struct {
	Msg       *sipmsg.SipMsg
	CallId    string
	LocalTag  string
	RemoteTag string
	Transport string
  ToNumber  string
  ToNoa string
  FromNumber string
  FromNoa string
  
}

func (d *BData) fill(msg *sipmsg.SipMsg) {
	d.Msg = msg
	d.CallId = msg.Transaction.Call.CallId
	d.LocalTag = msg.Transaction.LocalTag
	d.RemoteTag = msg.Transaction.RemoteTag
	//
}

func (d *BData) fillDefault(ci *utils.CallItem) {
	d.CallId = utils.RandString(10)
	d.LocalTag = utils.RandString(10)
  if len(ci.Noa)>0 { d.ToNoa=ci.Noa} else { d.ToNoa="2" }
  if len(ci.RLcallParty.Noa)>0 { d.FromNoa=ci.RLcallParty.Noa}else{  d.FromNoa="2"}
}

func GetOrDefault(m map[string]string, key,def string) string {
  val,ok:=m[key]
  if ok { return val }
  return def
}


func addHeader(h sipmsg.MsgHeaders, name string, val string) {
	h[name] = append(h[name], val)
}

func build(prev *sipmsg.SipMsg, ci *utils.CallItem, req int, remote string) (b *Builder) {
	// for debugging and error messages
	cp := ci.RLcallParty
	st := cp.RLsingleTest
	ts := st.RLtestSuite
	fn := ts.RLfileName
	//
	b = new(Builder)
	msg := new(sipmsg.SipMsg)
	b.msg = msg
	msg.Prev = prev
	msg.SipType = req
	data := new(BData)
	var err error
	if req >= 100 {
		for p := prev; p != nil; p = p.Prev {
			if p.Prev == nil || p.SipType < 100 {
				msg.Req = p
				break
			}
		}
	}
	// adding the transport
	trParts := strings.Split(remote, "/")
	if len(trParts) < 2 {
		log.Fatal("Remote wrong specified: " + remote)
	}
	data.Transport = trParts[1]
	// filling other data
  data.fillDefault(ci)
	var call *sipmsg.SipCall
	var trans *sipmsg.SipTransaction
	if prev != nil {
		data.fill(prev)
		trans = prev.Transaction
		call = trans.Call
		// TODO when do we have new transactions?
	} else {
		call = new(sipmsg.SipCall)
		call.CallId = utils.RandString(10)
		trans = new(sipmsg.SipTransaction)
		trans.Call = call
	}
	msg.Transaction = trans
	templ := template.New("SipMsg")
	templ = templ.Funcs(funcMap)
	if len(ci.Headers) > 0 {
		templ, err = templ.Parse(ci.Headers)
		if err != nil {
			log.Printf("wrong template in %s: %s/%s/%s template below\n%s\n", fn, ts.Name, st.Name, cp.Number, ci.Headers)
			log.Fatal(err)
		}
		bb := new(bytes.Buffer)
		err = templ.Execute(bb, data)
		if err != nil {
			log.Printf("error in %s: %s/%s/%s template below\n%s\n", fn, ts.Name, st.Name, cp.Number, ci.Headers)
			log.Fatal(err)
		}
		str := bb.String()
		err = msg.Retrieve(str)
		if err != nil {
			log.Printf("error in %s: %s/%s/%s expanded template below\n%s\n", fn, ts.Name, st.Name, cp.Number, str)
			log.Fatal(err)
		}
	}
	b.templ = templ
	b.data = data
	// completing the message
	// trial
	// TODO change from short to long form
	val, ok := msg.Headers["Call-ID"]
	if ok {
		if len(val) > 1 {
			log.Printf("too many Call-ID's in %s: %s/%s/%s", fn, ts.Name, st.Name, cp.Number)
			os.Exit(2)
		}
		call.CallId = val[0]
	} else {
		addHeader(msg.Headers, "Call-ID", call.CallId)
	}
	val, ok = msg.Headers["CSeq"]
	if ok {
		if len(val) > 1 {
			log.Printf("too many CSeq's in %s: %s/%s/%s", fn, ts.Name, st.Name, cp.Number)
			os.Exit(2)
		}
		csf := strings.Fields(val[0])
		v, err := strconv.Atoi(csf[0])
		if err != nil {
			log.Printf("CSeq error in %s: %s/%s/%s", fn, ts.Name, st.Name, cp.Number)
			log.Fatal(err)
		}
		call.CallSeq = v
	} else {
		switch req {
		case sipmsg.ReqInvite, sipmsg.ReqPrack:
			call.CallSeq++
		default:
		}
		var v string
		if prev != nil {
			v = prev.Headers["CSeq"][0]
		} else {
			v = fmt.Sprintf("%d %s", call.CallSeq, sipmsg.SipT2S(req))
		}
		addHeader(msg.Headers, "CSeq", v)
	}
	val, ok = msg.Headers["Max-Forwards"]
	if ok {
		if len(val) > 1 {
			log.Printf("too many Call-ID's in %s: %s/%s/%s", fn, ts.Name, st.Name, cp.Number)
			os.Exit(2)
		}
	} else {
		addHeader(msg.Headers, "Max-Forwards", "70")
	}
	val, ok = msg.Headers["From"]
	if ok {
	} else {
	}
	return
}
