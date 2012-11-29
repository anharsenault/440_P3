package main

import (
  "flag"
  "fmt"
  "os"
  "strings"
  "P3-f12/official/lsp12"
  "P3-f12/official/lspnet"
  "P3-f12/official/lsplog"
)

/**
 * Echo client main routine.
 * Wait for user input and forward to the echoserver.
 */
func runclient(cli *lsp12.LspClient) {
  var str string

  for {
    // read next token from input
    fmt.Printf("CLI-SRV: ")
    _, err := fmt.Scan(&str)
    if err != nil || strings.EqualFold(str, "quit") {
      cli.Close()
      return
    }

    // send to server
    werr := cli.Write([]byte(str))
    if werr != nil {
      fmt.Printf("Lost contact with server on write: %s\n", werr.Error())
      return
    }

    // read from server
    payload, rerr := cli.Read()
    if rerr != nil {
      fmt.Printf("Lost contact with server on read: %s\n", rerr.Error())
      return
    }

    fmt.Printf("SRV-CLI: [%s]\n", string(payload))
  }

  fmt.Printf("Exiting.\n")
  cli.Close()
}

func main() {
  var ihelp *bool = flag.Bool("h", false, "Show help information")
  var iport *int = flag.Int("p", 55455, "Port number")
  var ihost *string = flag.String("H", "localhost", "Host address")
  var iverb *int = flag.Int("v", 1, "Verbosity (0-6)")
  var irdrop *int = flag.Int("r", 0, "Network read packet drop percentage")
  var iwdrop *int = flag.Int("w", 0, "Network write packet drop percentage")
  var elim *int = flag.Int("k", 5, "Epoch limit")
  var ems *int = flag.Int("d", 2000, "Epoch duration (millisecconds)")

  flag.Parse()
  if *ihelp {
    flag.Usage()
    os.Exit(0)
  }

  if flag.NArg() > 0 {
    // Look for host:port on command line
    ok := true
    fields := strings.Split(flag.Arg(0), ":")
    ok = ok && len(fields) == 2
    if ok {
      *ihost = fields[0]
      n, err := fmt.Sscanf(fields[1], "%d", iport)
      ok = ok && n == 1 && err == nil
    }
    if !ok {
      flag.Usage()
      os.Exit(0)
    }
  }

  params := &lsp12.LspParams{*elim,*ems}

  lsplog.SetVerbose(*iverb)
  lspnet.SetReadDropPercent(*irdrop)
  lspnet.SetWriteDropPercent(*iwdrop)

  hostport := fmt.Sprintf("%s:%v", *ihost, *iport)
  fmt.Printf("Connecting to server at %s\n", hostport)

  cli, err := lsp12.NewLspClient(hostport, params)
  if lsplog.CheckReport(1, err) {
    return
  }

  runclient(cli)
}
