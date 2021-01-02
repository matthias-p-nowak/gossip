package utils

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	// The read in data, shouldn't be modified inside the program
	Suites []*TestSuite
)

type CallStep struct {
	// Alias for referential
	Alias string `yaml:"alias"`
	// Allow header, several strings
	Allow string `yaml:"allow"`
	// Body a template
	Body string `yaml:"body"`
	// a programmed pause
	Delay string `yaml:"delay"`
	// Headers is a template for additional and require headers, other required headers will be added
	Headers string `yaml:"headers"`
	// Noa = nature of address for the outgoing call
	Noa string `yaml:"noa"`
	// Out the request or response for this message
	Out string `yaml:"out"`
	// Previous tags the previous related message
	Previous string `yaml:"previous"`
	// Required - several strings
	Required string `yaml:"required"`
	// Sdp indicates what sdp body should be send, several strings
	Sdp string `yaml:"sdp"`
	// Supported header, several strings
	Supported string `yaml:"supported"`
	// Tags influences what other actions should be taken
	Tags string `yaml:"tags"`
	// To is just the number to call
	To string `yaml:"to"`
	// ################### prepared stuff
	AllowTags     map[string]bool
	SupportedTags map[string]bool
	RequiredTags  map[string]bool
	SdpTags       map[string]bool
	TagsTags      map[string]bool
	// reverse link
	RLcallParty *CallParty
}

// getTags: auxiliary function returning a set of found strings
func getTags(str string) (m map[string]bool) {
	m = make(map[string]bool)
	for _, s := range strings.Fields(str) {
		m[s] = true
	}
	return
}

// prepares Tags for easier access
func (ci *CallStep) prepare() {
	ci.AllowTags = getTags(ci.Allow)
	ci.SupportedTags = getTags(ci.Supported)
	ci.RequiredTags = getTags(ci.Required)
	ci.SdpTags = getTags(ci.Sdp)
	ci.TagsTags = getTags(ci.Tags)
}

// CallParty describes one leg in a test call
type CallParty struct {
	// Number is the parties number, aka own number
	Number string `yaml:"number"`
	// Noa - nature of address for outgoing messages
	Noa string `yaml:"noa"`
	// Steps, the steps in that call leg
	Steps []*CallStep `yaml:"steps"`
	// reverse link
	RLsingleTest *SingleTest
}

// SingleTest describes one single test wit several call parties
type SingleTest struct {
	// Name for reports
	Name        string       `yaml:"name"`
	CallParties []*CallParty `yaml:"calls"`
	// reverse link
	RLtestSuite *TestSuite
}

// TestSuite comprises a set of tests
type TestSuite struct {
	Name  string        `yaml:"suite"`
	Tests []*SingleTest `yaml:"tests"`
	// reverse link
	RLfileName string
}

// ReadSpec reads files which should contain a TestSuite
func ReadSpec(fn string, info os.FileInfo, err error) error {
	if err != nil {
		log.Println(err)
		// cannot correct this
		return err
	}
	// skip directories
	if !info.Mode().IsRegular() {
		log.Println("skipping " + fn)
		return nil
	}
	log.Println("reading file " + fn)
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatal(err)
	}
	ts := new(TestSuite)
	err = yaml.Unmarshal(data, ts)
	if err != nil {
		log.Fatal(err)
	}
	ts.RLfileName = fn
	// have the test suite
	// going through the tests and add back links and prepare
	for _, st := range ts.Tests {
		st.RLtestSuite = ts
		for _, cp := range st.CallParties {
			cp.RLsingleTest = st
			for _, ci := range cp.Steps {
				ci.RLcallParty = cp
				ci.prepare()
			}
		}
	}
	Suites = append(Suites, ts)
	return err
}

// GetAllTests is a generator function sending single test over the output channel
func GetAllTests(cfg *Config) chan *SingleTest {
	ch := make(chan *SingleTest)
	// the next is an anonymous function run as a goroutine
	go func() {
		for i := 0; i < cfg.Loops; i++ {
			if cfg.Continuous {
				i = 0
			}
			for _, ts := range Suites {
				for _, gt := range ts.Tests {
					ch <- gt
				}
			}
		}
		// a closed channel ends a range channel constuct
		close(ch)
	}()
	// returning the channel
	return ch
}
