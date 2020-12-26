package utils

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
  "strings"
)

var (
	Suites []*TestSuite
)

type CallItem struct {
  Alias string `yaml:"alias"`
  Previous string `yaml:"previous"`
  Out string `yaml:"out"`
  To string `yaml:"to"`
  Noa string `yaml:"noa"`
  Headers string `yaml:"headers"`
  Allow string `yaml:"allow"`
  Supported string `yaml:"supported"`
  Required string `yaml:"required"`
  Sdp string `yaml:"sdp"`
  Tags string `yaml:"tags"`
  Templates map[string]string `yaml:"templates"`
  // prepared stuff
  AllowTags map[string]bool
  SupportedTags map[string]bool
  RequiredTags map[string]bool
  SdpTags map[string]bool
  TagsTags map[string]bool
  // reverse link
  RLcallParty *CallParty
}

func getTags(str string)(m map[string]bool){
  m=make(map[string]bool)
  for _,s:= range strings.Fields(str){
    m[s]=true
  }
  return
}

func (ci *CallItem) prepare(){
  ci.AllowTags=getTags(ci.Allow)
  ci.SupportedTags=getTags(ci.Supported)
  ci.RequiredTags=getTags(ci.Required)
  ci.SdpTags=getTags(ci.Sdp)
  ci.TagsTags=getTags(ci.Tags)
}

type CallParty struct {
  Number string  `yaml:"number"`  
  Noa string `yaml:"noa"`
  Steps []*CallItem `yaml:"steps"`
  // reverse link
  RLsingleTest *SingleTest
}

// SingleTest describes one single test
type SingleTest struct {
	Name string `yaml:"name"`
  CallParties []*CallParty `yaml:"calls"`
  // reverse link
  RLtestSuite *TestSuite
}


// TestSuite comprises a set of tests
type TestSuite struct {
	Name  string      `yaml:"suite"`
	Tests []*SingleTest `yaml:"tests"`
  // reverse link
  RLfileName string
}

func ReadSpec(fn string, info os.FileInfo, err error) error {
  
	if err != nil {
		log.Println(err)
		return err
	}
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
  ts.RLfileName=fn
  for _,st:= range ts.Tests {
    st.RLtestSuite=ts
    for _,cp:= range st.CallParties {
      cp.RLsingleTest = st
      for _,ci:= range cp.Steps {
        ci.RLcallParty=cp
        ci.prepare()
      }
    }
  }
  Suites=append(Suites,ts)
	return err
}

func GetAllTests(cfg *Config)(chan *SingleTest){
  ch:=make(chan *SingleTest)
  go func(){
    for i := 0; i < cfg.Loops; i++ {
      if cfg.Continuous {
        i = 0
      }
      for _,ts := range Suites {
        for _,gt := range ts.Tests {
          ch <- gt
        }
      }
    }
    close(ch)
  }()
  return ch
}
