package utils

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

var (
	Suites []*TestSuite
)

type CallItem struct {
  Alias string `yaml:"alias"`
  Out string `yaml:"out"`
  To string `yaml:"to"`
}

type CallParty struct {
  Number string  `yaml:"number"`  
  Steps []*CallItem `yaml:"steps"`
}

// SingleTest describes one single test
type SingleTest struct {
	Name string `yaml:"name"`
  CallParties []*CallParty `yaml:"calls"`
}


// TestSuite comprises a set of tests
type TestSuite struct {
	Name  string      `yaml:"suite"`
	Tests []*SingleTest `yaml:"tests"`
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
