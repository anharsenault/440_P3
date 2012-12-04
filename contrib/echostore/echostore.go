package main

import (
  "flag"
  "fmt"
  "net"
  "net/http" 
  "net/rpc"
  "os"
  "strings"
  "sync"
  "P3-f12/contrib/echoproto"
)

var log []string
var peers []string

type Store struct {
  Id int
  Lock sync.Mutex

  Log []string
  Peers []*rpc.Client
  Primary bool
}

func NewStore(Id int) *Store {
  var svr Store

  svr.Id = Id
  svr.Log = nil
  svr.Peers = nil
  svr.Primary = false

  return &svr
}

func (svr *Store) info(fun string) {
  log.Printf("B%d %s: Primary = %d\n",
      svr.Id, fun, svr.Primary)
}

func (svr *Store) Stream(args *echoproto.Args, reply *echoproto.Reply) {
  var err error

  if !svr.Primary {
    return
  }

  for i := 0; i < len(svr.Peers); i++ {
    err = rpc.Call("Store.AppendLog", args, reply)

    if err != nil {
      log.Println("Backend storage server down.")
    }
  }
}

func (svr *Store) AppendLog(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()
  svr.Log = append(svr.Log, args.V)
  svr.Lock.Unlock()

  svr.Stream(args, reply)

  return nil
}

func append_log(str string) {
  log = append(log, str)
}

func upgrade(st_peers string) {
  peers = strings.Split(string(st_peers), ";")
  fmt.Printf("peers len: %d\n", len(peers))

  return
}

func runserver(cli *lsp12.LspClient) {
  var str string

  // register with master echoserver
  cli.Write(nil)

  for {
    // read from echoserver
    payload, rerr := cli.Read()
    if lsplog.CheckReport(1, rerr) {
      fmt.Printf("Echo server has died.\n")
      return
    }
    
    str = string(payload)
    lsplog.Vlogf(3, "Storage server received '%s'.\n", str)

    switch {
    case str[0] == 'A':
      append_log(strings.Fields(str)[1])

    case str[0] == 'F':
      fmt.Printf("Dumping log.\n")
      for i := 0; i < len(log); i++ {
        fmt.Printf("%s ", log[i])
      }
      fmt.Printf("\n")

    case str[0] == 'U':
      upgrade(strings.Fields(str)[1])

    default:
      fmt.Printf("Invalid command.\n")
    }
  }
}

func main() {
  var ihelp *bool = flag.Bool("h", false, "Print help information")
  var iport *int = flag.Int("p", 0, "Port number")
  var iverb *int = flag.Int("v", 1, "Verbosity (0-6)")
  var idrop *int = flag.Int("r", 0, "Network packet drop percentage")
  var elim *int = flag.Int("k", 5, "Epoch limit")
  var ems *int = flag.Int("d", 2000, "Epoch duration (millisecconds)")

  var master *string = flag.String("H", "localhost:55455", "echo server address")

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

  runtime.GOMAXPROCS(8)

  // register a server object to the RPC interface defined above
  svr = NewStore(*id)
  svr.Lock.Lock()

  rpc.Register(svr)
  rpc.HandleHTTP()

  params := &lsp12.LspParams{*elim,*ems}

  cli, err := lsp12.NewLspClient((*master), params)
  if lsplog.CheckReport(1, err) {
    fmt.Printf("Failed to create storage server.\n", err.Error())
  }

  runserver(cli)
}
