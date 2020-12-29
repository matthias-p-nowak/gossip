package tester

import (
	"bytes"
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
	Msg        *sipmsg.SipMsg
	CallId     string
	CSeq       int
	LocalTag   string
	RemoteTag  string
	Transport  string
	ToNumber   string
	ToNoa      string
	FromNumber string
	FromNoa    string
	Request    string
	Localhost  string
}

func (d *BData) fill(ci *utils.CallItem, msg *sipmsg.SipMsg) {
	d.Msg = msg
	if msg != nil {
		d.CallId = msg.Transaction.Call.CallId
		d.LocalTag = msg.Transaction.LocalTag
		d.RemoteTag = msg.Transaction.RemoteTag
	} else {
		d.CallId = utils.RandString(10)
		d.LocalTag = utils.RandString(10)
	}
	//
	if len(ci.Noa) > 0 {
		d.ToNoa = ci.Noa
	} else {
		d.ToNoa = "2"
	}
	if len(ci.RLcallParty.Noa) > 0 {
		d.FromNoa = ci.RLcallParty.Noa
	} else {
		d.FromNoa = "2"
	}
}

func GetOrDefault(m map[string]string, key, def string) string {
	val, ok := m[key]
	if ok {
		return val
	}
	return def
}

func addHeader(h sipmsg.MsgHeaders, name string, val string) {
	h[name] = append(h[name], val)
}

func LogError(ci *utils.CallItem, reason, msg string, err error) {
	cp := ci.RLcallParty
	st := cp.RLsingleTest
	ts := st.RLtestSuite
	fn := ts.RLfileName
	log.Printf("%s in %s: %s/%s/%s \n%s\n", reason, fn, ts.Name, st.Name, cp.Number, msg)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(2)
}

func (b *Builder) fromTemplate(ci *utils.CallItem, id, t string) string {
	str, ok := ci.Templates[id]
	if !ok {
		str = t
	}
	templ, err := b.templ.Parse(str)
	if err != nil {
		LogError(ci, "wrong template", str, err)
	}
	bb := new(bytes.Buffer)
	err = templ.Execute(bb, b.data)
	if err != nil {
		LogError(ci, "not executing ", str, err)
	}
	return bb.String()
}

func buildSip(prev *sipmsg.SipMsg, ci *utils.CallItem, req int, remote string) (b *Builder) {
	var call *sipmsg.SipCall
	var trans *sipmsg.SipTransaction
	b = new(Builder)
	msg := new(sipmsg.SipMsg)
	b.msg = msg
	msg.Prev = prev
	msg.SipType = req
	data := new(BData)
	b.data = data
	templ := template.New("SipMsg")
	templ = templ.Funcs(funcMap)
	b.templ = templ
	b.data = data
	data.Request = sipmsg.SipT2S(req)
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
	if prev != nil {
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
	switch req {
	case sipmsg.ReqInvite, sipmsg.ReqPrack:
		call.CallSeq++
	default:
	}
	data.fill(ci, prev)
	if len(ci.Headers) > 0 {
		str := b.fromTemplate(ci, "headers", ci.Headers)
		// store provided headers
		err = msg.Retrieve(str)
		if err != nil {
			LogError(ci, "couldn't get headers from", str, err)
		}
	}
	// completing the message
	// trial
	// TODO change from short to long form
	// ##### Call-ID
	val, ok := msg.Headers["Call-ID"]
	if ok {
		if len(val) > 1 {
			LogError(ci, "too many Call-ID's", val[0], nil)
		}
		call.CallId = val[0]
	} else {
		t := "{{ .CallId }}"
		t = b.fromTemplate(ci, "callid", t)
		addHeader(msg.Headers, "Call-ID", t)
	}
	// ##### CSeq
	val, ok = msg.Headers["CSeq"]
	if ok {
		if len(val) > 1 {
			LogError(ci, "too many CSeq's", val[0], nil)
		}
		csf := strings.Fields(val[0])
		v, err := strconv.Atoi(csf[0])
		if err != nil {
			LogError(ci, "CSeq error", val[0], err)
		}
		call.CallSeq = v
	} else {
		var v string
		if prev != nil {
			v = prev.Headers["CSeq"][0]
		} else {
			v = "{{.CSeq}} {{ .Request }}"
			v = b.fromTemplate(ci, "cseq", v)
		}
		addHeader(msg.Headers, "CSeq", v)
	}
	// ##### Max-Forwards
	val, ok = msg.Headers["Max-Forwards"]
	if ok {
		if len(val) > 1 {
			LogError(ci, "too many Max-Forwards's", val[0], nil)
		}
	} else {
		addHeader(msg.Headers, "Max-Forwards", "70")
	}
	// ##### From
	val, ok = msg.Headers["From"]
	if ok {
		if len(val) > 1 {
			LogError(ci, "too many From's", val[0], nil)
		}
		// TODO regexp/grep the "from" field
	} else {
		var v string
		if prev != nil {
			v = trans.Local
		} else {
			if req < 100 {
				v = "<{{.FromNumber}}@{{.Localhost}};noa={{.FromNoa}}>;tag={{.LocalTag}}"
				v = b.fromTemplate(ci, "from", v)
			} else {
				LogError(ci, "can't start with a response", "-no from-", nil)
			}
		}
		addHeader(msg.Headers, "From", v)
	}
	// ##### To
	val, ok = msg.Headers["To"]
	if ok {
		if len(val) > 1 {
			LogError(ci, "too many To's", val[0], nil)
		}
		// TODO regexp/grep the "to" field
	} else {
		var v string
		if prev != nil {
			v = trans.Remote
		} else {
			if req < 100 {
				v = "<{{.ToNumber}}@{{.Transport}};noa={{.ToNoa}}>"
				v = b.fromTemplate(ci, "to", v)
			} else {
				LogError(ci, "can't start with a response", "-no to-", nil)
			}
		}
		addHeader(msg.Headers, "To", v)
	}
	return
}

func (b *Builder) createSDP(ci *utils.CallItem) {
	addHeader(b.msg.Headers, "Content-Type", "application/sdp")
	h := ci.SdpTags
	if h["offer"] {
		if h["dummy"] {
			var l []string
			l = append(l, "v=0")
			l = append(l, "o=gossip 1 1 IN IP4 127.0.0.1")
			l = append(l, "s=gossip dummy session")
			l = append(l, "c=IN IP4 127.0.0.1")
			l = append(l, "t=0 0")
			l = append(l, "m=audio 63999 RTP/AVP 0")
			l = append(l, "a=rtpmap:0 PCMU/8000")
			b.msg.BodyList = l
		}
	}
}

func (b *Builder) buildItem() *sipmsg.Item {
	item := new(sipmsg.Item)
  item.Msg=b.msg
	return item
}
