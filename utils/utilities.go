package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	//"os"
	//"io"
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
	return
}

