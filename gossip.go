package main

import (
  "log"
)

func main(){
  log.SetFlags(log.LstdFlags | log.Lshortfile)
  defer log.Println("all done")
  log.Println(" gossip started")
  // ----------
}
