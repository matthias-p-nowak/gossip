// Gossip runs sip tests
package main

//go:generate go run scripts/go-bin.go -o snippets.go snippets

import (
  "log"
  "os"
  "flag"
  "gossip/utils"
  "path/filepath"
  "syscall"
  "os/signal"
  "gossip/tester"
)

var (
  cfg *utils.Config
  running=true
)

func handleSignals(){
  signals := make(chan os.Signal, 1)
  signal.Notify(signals, syscall.SIGHUP,syscall.SIGQUIT,syscall.SIGTERM)
  for s := range signals {
    log.Println("Got signal:", s)
    running=false
  }
}

// main runs gossip
func main(){
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
  if len(*logFile)>0 {
    logF,err := os.OpenFile(*logFile,os.O_CREATE|os.O_WRONLY,0644)
    if err != nil { log.Fatal(err)}
    defer logF.Close()
    log.Printf("changing log to %s\n",*logFile)
    log.SetOutput(logF)    
  }
  // ----- reading the configuration
  log.Printf("reading config from %s\n",*cfgFile)
  cfg, err := utils.GetConfig(*cfgFile)
  if err != nil { log.Fatal(err) }
  // ----- reading the tests
  for _, arg := range flag.Args() {
    filepath.Walk(arg, utils.ReadSpec)
  }
  // ----- running all the tests
  utils.Limiter(cfg)
  for t:=range utils.GetAllTests(cfg) {
    tr:=tester.Create(t)
    go tr.Run()
  }
  utils.Wait()
}
