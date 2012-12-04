package echoclient

import (
  "net/rpc"
  "fmt"
/*
  "flag"
  "fmt"
  "log"
  "os"
  "strings"
  */
  "P3-f12/contrib/echoproto"
)

/**
 * Echo client main routine.
 * Wait for user input and forward to the echoserver.
 */

/*
func runclient(cli *rpc.Client) {
  var args echoproto.Args
  var reply echoproto.Reply
  var err, werr error

  for {
    // read next token from input
    fmt.Printf("CLI-SRV: ")
    _, err = fmt.Scan(&args.V)
    if err != nil || strings.EqualFold(args.V, "quit") {
      break
    }

    if strings.EqualFold(args.V, "%%") {
      werr = cli.Call("Server.FetchLog", args, &reply)
      if werr != nil {
        break
      }

      log.Println("Dumping contents of log.")
      for i := 0; i < len(reply.Data); i++ {
        fmt.Printf("%s ", reply.Data[i])
      }
      fmt.Printf("\n")
    } else {
      werr = cli.Call("Server.AppendLog", args, &reply)
      if err != nil {
        break
      }

      fmt.Printf("SRV-CLI: [%s]\n", reply.Data[0])
    }
  }

  if werr != nil {
    log.Printf("Lost contact with server: %s\n", werr.Error())
  }

  log.Println("Exiting.")
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
  if err != nil {
    log.Fatalln("rpc.DialHTTP() error: %s", err.Error())
  }

  runclient(cli)
}*/

type EchoClient struct {
  rpcAddr string
  rpcSvr *rpc.Client
}

func NewEchoClient(addr string) (*EchoClient, error) {
  var cli EchoClient

  client, err := rpc.DialHTTP("tcp", addr)
  if err != nil {
    return nil, err
  }

  cli.rpcAddr = addr
  cli.rpcSvr = client

  return &cli, nil
}

func (cli *EchoClient)Send(str string) (string, error) {
  var args echoproto.Args
  var reply echoproto.Reply

  args.V = str
  args.N = 1

  werr := cli.rpcSvr.Call("Server.AppendLog", args, &reply)
  if werr != nil {
    fmt.Printf("Lost contact with server on write: %s\n", werr.Error())
    return "", werr
  }

  fmt.Printf("echo client send replay %d %s\n", reply.Response, reply.Data[0])

  return reply.Data[0], nil
}

func (cli *EchoClient)Fetch() ([]string, error) {
  var args echoproto.Args
  var reply echoproto.Reply

  werr := cli.rpcSvr.Call("Server.FetchLog", args, &reply)
  if werr != nil {
    fmt.Printf("Lost contact with server on write: %s\n", werr.Error())
    return nil, werr
  }

  return reply.Data, nil
}
