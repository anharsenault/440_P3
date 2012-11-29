package main

import (
  "flag"
  "fmt"
  "os"
  "P3-f12/official/lsp12"
  "P3-f12/official/lspnet"
  "P3-f12/official/lsplog"
)

var st_info map[uint16]uint16
var log bool = false
var pri uint16

func append_log(svr *lsp12.LspServer, str string) {
  if !log {
    return
  }

  for id := range(st_info) {
    payload := []byte(fmt.Sprintf("A %s", str))
    svr.Write(id, payload)
  }
}

func fetch_log(svr *lsp12.LspServer) {
  if !log {
    return
  }

  lsplog.Vlogf(4, "Fetching log from primary backup.\n")

  payload := []byte("F")
  svr.Write(pri, payload)
}

/**
 * Basic echoserver routine.
 * Wait for client connection and echo the string.
 */
func runserver(svr *lsp12.LspServer) {
  for {
    // Read from client
    id, payload, rerr := svr.Read()
    if lsplog.CheckReport(1, rerr) {
      fmt.Printf("Connection %d has died.\n", id)

      // storage server has died, switch primary
      if _, ok := st_info[id]; ok {
        lsplog.Vlogf(4, "Storage server %d has died.\n", id)

        delete(st_info, id)

        for id := range(st_info) {
          pri = id
          break
        }
      }

      fetch_log(svr)
    } else {
      s := string(payload)
      lsplog.Vlogf(6, "(C%d) Received: '%s'\n", id, s)

      append_log(svr, s)

      werr := svr.Write(id, payload)
      if lsplog.CheckReport(1, werr) {
        fmt.Printf("Connection %d. Write failed.\n", id)
      }
    }
  }
}

func main() {
  var ihelp *bool = flag.Bool("h", false, "Print help information")
  var iport *int = flag.Int("p", 55455, "Port number")
  var iverb *int = flag.Int("v", 1, "Verbosity (0-6)")
  var idrop *int = flag.Int("r", 0, "Network packet drop percentage")
  var elim *int = flag.Int("k", 5, "Epoch limit")
  var ems *int = flag.Int("d", 2000, "Epoch duration (millisecconds)")
  var st_num *int = flag.Int("S", 2, "Number of storage servers")

  lsplog.SetVerbose(*iverb)
  lspnet.SetWriteDropPercent(*idrop)

  flag.Parse()
  if *ihelp {
    flag.Usage()
    os.Exit(0)
  }

  var port int = *iport
  if flag.NArg() > 0 {
    nread, _ := fmt.Sscanf(flag.Arg(0), "%d", &port)
    if nread != 1 {
      flag.Usage()
      os.Exit(0)
    }
  }

  st_info = make(map[uint16]uint16)

  params := &lsp12.LspParams{*elim,*ems}

  // create master server on specified port
  svr, err := lsp12.NewLspServer(port, params)
  if lsplog.CheckReport(1, err) {
    fmt.Printf("Failed to create echoserver.\n")
  }

  if *st_num == 0 {
    runserver(svr)
  } else {
    log = true
  }

  // echoserver will wait for incoming join requests from storage servers
  for i := 0; i < *st_num; i++ {
    connID, st_host, _ := svr.Read()

    lsplog.Vlogf(4, "Received connection from storage server %s.\n", string(st_host))
    st_info[connID] = connID
    pri = connID
  }

  runserver(svr)
}
