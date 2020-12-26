package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
  "time"
	"math/rand"
	//"os"
	//"io"
)


var (
	// Alphabet comprises the symbols for creating random strings
	Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// Config contains the config data provided by YAML
type Config struct {
	Continuous bool     `yaml:"continuous"`
	Loops      int      `yaml:"loops"`
	Rate       int      `yaml:"rate"`
	Concurrent int32    `yaml:"concurrent"`
	Local      []string `yaml:"local"`
	Remote     []string `yaml:"remote"`
}

func (cfg *Config) init() {
	cfg.Continuous = false
	cfg.Loops = 1
	cfg.Rate = 1
	cfg.Concurrent = 1
	cfg.Local = []string{"udp/0.0.0.0:5065"}
	cfg.Remote = []string{"udp/localhost:5060"}
	t := time.Now().Unix()
	rand.Seed(t)

}

// GetConfig reads the content of <fn> and returns configuration
func GetConfig(fn string) (cfg *Config, err error) {
	cfg = new(Config)
	cfg.init()
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("ERROR: '%s' is not a valid config file.\n", fn)
		// io.Copy(os.Stdout, GetStored("snippets/gossip.cfg"))
		return
	}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		log.Fatal(err)
	}
  data,err=yaml.Marshal(cfg)
  fmt.Printf("using configuration:\n%s\n",string(data))
	return
}


// RandStrings returns a string of length <l>
func RandString(l int) string {
	aLen := len(Alphabet)
	bb := make([]byte, l)
	for i := 0; i < l; i++ {
		bb[i] = Alphabet[rand.Intn(aLen)]
	}
	return string(bb)
}
