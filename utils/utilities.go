package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	// Alphabet comprises the symbols for creating random strings
	Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	tRemote  int
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
	data, err = yaml.Marshal(cfg)
	if err != nil {
		log.Println("Marshalling error")
		return
	}
	fmt.Printf("using configuration:\n%s\n", string(data))
	for _, l := range cfg.Local {
		parts := strings.Split(l, "/")
		if len(parts) < 2 {
			log.Fatalf("local '%s' is not of the form <transport>/<host>:<port>, with transport being udp or tcp\n", l)
		}
	}
	for _, r := range cfg.Remote {
		parts := strings.Split(r, "/")
		if len(parts) < 2 {
			log.Fatalf("remote '%s' is not of the form <transport>/<host>:<port>, with transport being udp or tcp\n", r)
		}
	}
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

func (cfg *Config) GetTransport() (local, remote string) {
	tRemote = (tRemote + 1) % len(cfg.Remote)
	remote = cfg.Remote[tRemote]
	parts := strings.Split(remote, "/")
	transport := parts[0]
	for _, l := range cfg.Local {
		parts = strings.Split(l, "/")
		if parts[0] == transport {
			local = l
			return
		}
	}
	log.Fatal("could not find a valid local endpoint")
	return
}
