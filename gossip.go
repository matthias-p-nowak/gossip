// Gossip runs sip tests
package main

//go:generate go run scripts/go-bin.go -o snippets.go snippets

import (
  "log"
  "os"
  "flag"
  "gossip/utils"
)

var (
  cfg *utils.Config
)


// main runs gossip
func main(){
  log.SetFlags(log.LstdFlags | log.Lshortfile)
  defer log.Println("all done")
  log.Println("gossip started")
  cfgFile := flag.String("c", "gossip.cfg", "the configuration for gossip")
  logFile := flag.String("l", "", "writing to this logfile")
  flag.Parse()
  if len(*logFile)>0 {
    logF,err := os.OpenFile(*logFile,os.O_CREATE|os.O_WRONLY,0644)
    if err != nil { log.Fatal(err)}
    defer logF.Close()
    log.Printf("changing log to %s\n",*logFile)
    log.SetOutput(logF)    
  }
  //
  // ----------
  log.Printf("reading config from %s\n",*cfgFile)
  cfg, err := utils.GetConfig(*cfgFile)
  if err != nil { log.Fatal(err) }
  cfg=cfg
  
  for i, arg := range flag.Args() {
    log.Printf("i=%d, arg=%s\n",i,arg)
    //filepath.Walk(arg, parseTests)
  }
  
}
