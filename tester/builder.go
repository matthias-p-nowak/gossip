package tester

import (
	"bytes"
	"gossip/sipmsg"
	"gossip/utils"
	"log"
	"strconv"
	"strings"
	"text/template"
)

var (
	// is a map of function
	funcMap template.FuncMap
)

// automatically called
func init() {
	funcMap = make(template.FuncMap)
	funcMap["RandStr"] = utils.RandString
	// TODO add more useful functions here
}

// Builder for a SIP message
type Builder struct {
	templ *template.Template
	msg   *sipmsg.SipMsg
	data  *BData
	step  *utils.CallStep
}

// BData is for filling in templates
type BData struct {
	CallId     string
	CSeq       int
	FromNoa    string
	FromNumber string
	Localhost  string
	LocalSide  string
	LocalTag   string
	Msg        *sipmsg.SipMsg
	RemoteSide string
	RemoteTag  string
	Request    string
	ToNoa      string
	ToNumber   string
	Transport  string
}

// fills the data structure with information provided or from previous
func (d *BData) fill(ci *utils.CallStep, msg *sipmsg.SipMsg, tester *Tester) {
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
	d.FromNumber = ci.RLcallParty.Number
	if len(ci.To) > 0 {
		d.ToNumber = ci.To
	} else {
		log.Fatal("### not yet implemented")
	}
	d.LocalSide = tester.LocalParts[1]
	d.RemoteSide = tester.RemoteParts[1]
	d.Transport = tester.RemoteParts[0]
	strs := strings.Split(d.LocalSide, ":")
	d.Localhost = strs[0]
}

// aux function for returning default values when values are missing
func GetOrDefault(m map[string]string, key, def string) string {
	val, ok := m[key]
	if ok {
		return val
	}
	return def
}

// aux function simplifying adding header strings
func addHeader(h sipmsg.MsgHeaders, name string, val string) {
	h[name] = append(h[name], val)
}

// LogError add a few details, so one can identify the error more quickly
func LogError(step *utils.CallStep, reason, msg string, err error) {
	cp := step.RLcallParty
	st := cp.RLsingleTest
	ts := st.RLtestSuite
	fn := ts.RLfileName
	log.Fatalf("%s in %s: %s/%s/%s \n%s\n%s\n", reason, fn, ts.Name, st.Name, cp.Number, msg, err)
}

// fromTemplate uses the collected data to fill the template
func (b *Builder) fromTemplate(t string) string {
	templ, err := b.templ.Parse(t)
	if err != nil {
		LogError(b.step, "wrong template", t, err)
	}
	bb := new(bytes.Buffer)
	err = templ.Execute(bb, b.data)
	if err != nil {
		LogError(b.step, "not executing ", t, err)
	}
	return bb.String()
}

func newBuilder() (b *Builder) {
	b = new(Builder)
	b.data = new(BData)
	templ := template.New("SipMsg")
	templ = templ.Funcs(funcMap)
	b.templ = templ
	return
}

// builder creates a new Builder
func builder(prev *sipmsg.SipMsg, step *utils.CallStep, req int, tester *Tester) (b *Builder) {
	var call *sipmsg.SipCall
	var trans *sipmsg.SipTransaction
	var ok bool
	b = new(Builder)
	b.step = step
	msg := new(sipmsg.SipMsg)
	b.msg = msg
	msg.Prev = prev
	msg.SipType = req
	b.data.Request, ok = sipmsg.Req2String[req]
	if !ok {
		LogError(step, "couldn't find string for "+strconv.Itoa(req), "", nil)
	}
	var err error
	if req >= 100 {
		for p := prev; p != nil; p = p.Prev {
			if p.Prev == nil || p.SipType < 100 {
				msg.Req = p
				break
			}
		}
	}
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
	b.data.fill(step, prev, tester)
	if len(step.Headers) > 0 {
		str := b.fromTemplate(step.Headers)
		// store provided headers
		err = msg.Retrieve(str)
		if err != nil {
			LogError(step, "couldn't get headers from", str, err)
		}
	}
	// completing the message
	// trial
	// TODO change from short to long form
	// ##### Call-ID
	val, ok := msg.Headers["Call-ID"]
	if ok {
		if len(val) > 1 {
			LogError(step, "too many Call-ID's", val[0], nil)
		}
		call.CallId = val[0]
	} else {
		t := "{{ .CallId }}"
		t = b.fromTemplate(t)
		addHeader(msg.Headers, "Call-ID", t)
	}
	// ##### CSeq
	val, ok = msg.Headers["CSeq"]
	if ok {
		if len(val) > 1 {
			LogError(step, "too many CSeq's", val[0], nil)
		}
		csf := strings.Fields(val[0])
		v, err := strconv.Atoi(csf[0])
		if err != nil {
			LogError(step, "CSeq error", val[0], err)
		}
		call.CallSeq = v
	} else {
		var v string
		if prev != nil {
			v = prev.Headers["CSeq"][0]
		} else {
			v = "{{.CSeq}} {{ .Request }}"
			v = b.fromTemplate(v)
		}
		addHeader(msg.Headers, "CSeq", v)
	}
	// ##### Max-Forwards
	val, ok = msg.Headers["Max-Forwards"]
	if ok {
		if len(val) > 1 {
			LogError(step, "too many Max-Forwards's", val[0], nil)
		}
	} else {
		addHeader(msg.Headers, "Max-Forwards", "70")
	}
	// ##### From
	val, ok = msg.Headers["From"]
	if ok {
		if len(val) > 1 {
			LogError(step, "too many From's", val[0], nil)
		}
		// TODO regexp/grep the "from" field
	} else {
		var v string
		if prev != nil {
			v = trans.Local
		} else {
			if req < 100 {
				v = "<{{.FromNumber}}@{{.LocalSide}};noa={{.FromNoa}}>;tag={{.LocalTag}}"
				v = b.fromTemplate(v)
			} else {
				LogError(step, "can't start with a response", "-no from-", nil)
			}
		}
		addHeader(msg.Headers, "From", v)
	}
	// ##### To
	val, ok = msg.Headers["To"]
	if ok {
		if len(val) > 1 {
			LogError(step, "too many To's", val[0], nil)
		}
		// TODO regexp/grep the "to" field
	} else {
		var v string
		if prev != nil {
			v = trans.Remote
		} else {
			if req < 100 {
				v = "<{{.ToNumber}}@{{.RemoteSide}};noa={{.ToNoa}}>"
				v = b.fromTemplate(v)
			} else {
				LogError(step, "can't start with a response", "-no to-", nil)
			}
		}
		addHeader(msg.Headers, "To", v)
	}
	if req < 100 {
		v := "{{.Request}} sip:{{.ToNumber}}@{{.RemoteSide}};noa={{.ToNoa}} SIP/2.0"
		v = b.fromTemplate(v)
		msg.StartLine = v
	} else {
		log.Fatal("### not yet implemented")
	}
	return
}

// createSDP uses the SdpTags to find right version
func (b *Builder) createSDP(step *utils.CallStep) {
	addHeader(b.msg.Headers, "Content-Type", "application/sdp")
	h := step.SdpTags
	if h["offer"] {
		if h["dummy"] {
			str := `v=0
o=gossip 1 1 IN IP4 {{.Localhost}}
s=gossip dummy session
c=IN IP4 {{.Localhost}}
t=0 0
m=audio 63999 RTP/AVP 0
a=rtpmap:0 PCMU/8000`
			str = b.fromTemplate(str)
			b.msg.BodyList = strings.Split(str, "\n")
		}
	}
}

// getItem returns an Item based on the constructed builder
func (b *Builder) getItem() *sipmsg.Item {
	item := new(sipmsg.Item)
	item.Msg = b.msg
	// TODO add the other stuff
	return item
}
