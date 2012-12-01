package main

import (
  "flag"
  "fmt"
  "log"
  "net/rpc"
  "os"
  "strings"
  "P3-f12/official/lspnet"
  "P3-f12/official/lsplog"
)

type Args struct {
  Str string
  N int
}

type Reply struct {
  Data []string
}

/**
 * Echo client main routine.
 * Wait for user input and forward to the echoserver.
 */
func runclient(cli *rpc.Client) {
  var args Args
  var reply Reply

  for {
    // read next token from input
    fmt.Printf("CLI-SRV: ")
    _, err := fmt.Scan(&args.Str)
    if err != nil || strings.EqualFold(args.Str, "quit") {
      cli.Close()
      return
    }

    if strings.EqualFold(args.Str, "%%") {
      // send to server
      werr := cli.Call("Server.FetchLog", args, &reply)
      if werr != nil {
        fmt.Printf("Lost contact with server on write: %s\n", werr.Error())
        return
      }

      log.Println("Dumping log.")
      for i := 0; i < len(reply.Data); i++ {
        fmt.Printf("%s ", reply.Data[i])
      }
      fmt.Printf("\n")
    } else {
      // send to server
      werr := cli.Call("Server.AppendLog", args, &reply)
      if werr != nil {
        fmt.Printf("Lost contact with server on write: %s\n", werr.Error())
        return
      }

      fmt.Printf("SRV-CLI: [%s]\n", reply.Data[0])
    }
  }

  fmt.Printf("Exiting.\n")
  cli.Close()
}

func main() {
  var ihelp *bool = flag.Bool("h", false, "Show help information")
  var iport *int = flag.Int("p", 55455, "Port number")
  var ihost *string = flag.String("H", "localhost", "Host address")

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

  hostport := fmt.Sprintf("%s:%v", *ihost, *iport)
  log.Printf("Connecting to server: %s\n", hostport)

  cli, err := rpc.DialHTTP("tcp", hostport)
  if lsplog.CheckReport(1, err) {
    log.Fatalln("rpc.DialHTTP() error")
  }

  runclient(cli)
}
