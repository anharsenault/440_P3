package main

import (
  "flag"
  "fmt"
  "log"
  "net"
  "net/http" 
  "net/rpc"
  "os"
  "runtime"
  "sync"
  "time"
  "P3-f12/contrib/echoproto"
)

type Store struct {
  Id int
  Lock sync.Mutex

  Log []string
  Master *rpc.Client
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
    err = svr.Peers[i].Call("Store.AppendLog", args, reply)

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

func (svr *Store) FetchLog(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()
  reply.Data = svr.Log
  svr.Lock.Unlock()

  return nil
}

func (svr *Store) Upgrade(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()
  reply.Data = svr.Log
  svr.Lock.Unlock()

  return nil
}

/*
func upgrade(st_peers string) {
  peers = strings.Split(string(st_peers), ";")
  fmt.Printf("peers len: %d\n", len(peers))

  return
}
*/
/*
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
*/

func main() {
  var svr *Store
  var hostname string
  var block chan int
  var err error
  var args echoproto.Args
  var reply echoproto.Reply

  var ihelp *bool = flag.Bool("h", false, "Print help information")
  var iport *int = flag.Int("p", 0, "Port number")
  var master *string = flag.String("H", "localhost:55455", "echo server address")
  var id *int = flag.Int("i", 0, "Id number")

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

  hostname = fmt.Sprintf(":%d", *iport)

  // open a listening socket on a specified port
  conn, err := net.Listen("tcp", hostname)
  if err != nil {
    log.Fatalln("net.Listen() error:", err.Error())
  }

  go http.Serve(conn, nil)

  svr.Master, err = rpc.DialHTTP("tcp", *master)

  for j := 0; (err != nil) && (j < echoproto.N_ATTEMPTS); j++ {
    log.Println("rpc.DialHTTP() failed, retrying")

    time.Sleep(1000 * time.Millisecond)
    svr.Master, err = rpc.DialHTTP("tcp", *master)
  }

  if err != nil {
    log.Fatalln("rpc.DialHTTP() error:", err.Error())
  }

  svr.Master.Call("Server.Register", args, &reply)

  svr.Lock.Unlock()

  block = make(chan int)
  <- block
}
