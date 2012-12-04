package main

import (
  "flag"
  "fmt"
  "log"
  "net"
  "net/http"
  "net/rpc"
  "os"
  //"strings"
  "sync"
  "P3-f12/contrib/echoproto"
)

type Server struct {
  // synchronize access to local variables
  Lock sync.Mutex
  Id int
  Log []string

  // Lamport timestamp
  Time int

  N_high int
  N_accept int
  V_accept string
}

func NewServer(Id int) *Server {
  var svr Server

  svr.Id = Id
  svr.Log = nil
  svr.Time = 0
  svr.N_high = Id
  svr.N_accept = -1
  svr.V_accept = ""

  return &svr
}

func (svr *Server) AppendLog(args *echoproto.Args, reply *echoproto.Reply) error {
  // initiate a commit to the log
  svr.Lock.Lock()

  // increment the Lamport timestamp and append the string to the log
  svr.Time++
  svr.Log = append(svr.Log, args.Str)

  // respond with the same string
  reply.Data = nil
  reply.Data = append(reply.Data, args.Str)

  svr.Lock.Unlock()

  return nil
}

func (svr *Server) FetchLog(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()

  // increment the Lamport timestamp and copy the current version of the log
  svr.Time++
  reply.Data = svr.Log

  svr.Lock.Unlock()

  return nil
}

/**
 * RPC interface for the Paxos distributed agreement algorithm.
 **/
func (svr *Server) Prepare(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()

  if args.N >= svr.N_high {
    svr.N_high = args.N
    reply.Answer = echoproto.PREPARE_OK
  } else {
    reply.Answer = echoproto.PREPARE_REJECT
  }

  svr.Lock.Unlock()
  return nil
}

func (svr *Server) Accept(args *echoproto.Args, reply *echoproto.Reply) error {
  return nil
}

func (svr *Server) Commit(args *echoproto.Args, reply *echoproto.Reply) error {
  return nil
}

func main() {
  var ihelp *bool = flag.Bool("h", false, "Print help information")
  var iport *int = flag.Int("p", 55455, "Port number")
  /*
  var paxos *string = flag.String("P", "",
      "Hostnames of all other paxos nodes")
  */

  flag.Parse()
  if *ihelp {
    flag.Usage()
    os.Exit(0)
  }
/*
  var clients []*rpc.Client = nil
  hosts := strings.Split(*paxos, ",")
  for i := 0; i < len(hosts); i++ {
    cli, err := rpc.DialHTTP("tcp", hosts[i])
    if err != nil {
      log.Fatalln("rpc.DialHTTP() error: %s", err.Error())
    }

    clients = append(clients, cli)
  }
*/
  var port int = *iport
  if flag.NArg() > 0 {
    nread, _ := fmt.Sscanf(flag.Arg(0), "%d", &port)
    if nread != 1 {
      flag.Usage()
      os.Exit(0)
    }
  }

  // register a server object to the RPC interface defined above
  var svr_rpc *Server = new(Server)

  rpc.Register(svr_rpc)
  rpc.HandleHTTP()

  // open a listening socket on a specified port
  l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
  if err != nil {
    log.Fatalln("net.Listen() error: %s", err.Error())
  }
  fmt.Printf("listen on port %d\n", port)

  http.Serve(l, nil)
}
