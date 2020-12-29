// Gossip runs sip tests
package main

//go:generate go run scripts/go-bin.go -o snippets.go snippets

import (
	"flag"
	"gossip/infra"
	"gossip/tester"
	"gossip/utils"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var (
	cfg *utils.Config
	// False will abort
	Running = true
)

func handleSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	for s := range signals {
		log.Println("Got signal:", s)
		Running = false
		// stopping all tests
		tester.Running = false
	}
}

// main runs gossip
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	defer log.Println("all done")
	// ----- start signal handling
	go handleSignals()
	// ----- arguments
	log.Println("gossip started")
	cfgFile := flag.String("c", "gossip.cfg", "the configuration for gossip")
	logFile := flag.String("l", "", "writing to this logfile")
	flag.Parse()
	// ----- handling log redirection
	if len(*logFile) > 0 {
		logF, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer logF.Close()
		log.Printf("changing log to %s\n", *logFile)
		log.SetOutput(logF)
	}
	// ----- reading the configuration
	log.Printf("reading config from %s\n", *cfgFile)
	cfg, err := utils.GetConfig(*cfgFile)
	if err != nil {
		log.Fatal(err)
	}
	// ----- reading the tests
	for _, arg := range flag.Args() {
		filepath.Walk(arg, utils.ReadSpec)
	}
	// ----- setting up providers
	for _, provider := range cfg.Local {
		infra.NewProvider(provider)
	}
	// ----- running all the tests
	utils.Limiter(cfg)
	for t := range utils.GetAllTests(cfg) {
		if !Running {
			break
		}
		tr := tester.Create(t, cfg)
		go tr.Run()
	}
	utils.Wait()
	infra.CloseProviders()
}
