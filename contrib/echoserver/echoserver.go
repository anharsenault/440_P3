package main

import(
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

type Server struct {
  // synchronize access to local variables
  Lock sync.Mutex

  Id int
  Log []string

  // Lamport timestamp
  Time int

  F int
  N_high int
  N_accept int
  V_accept string

  Peers []*rpc.Client

  Store []string
  Primary *rpc.Client
  Backend []*rpc.Client
}

func NewServer(Id, F int) *Server {
  var svr Server

  svr.Id = Id
  svr.Log = nil
  svr.Time = Id

  svr.F = F
  svr.N_high = Id
  svr.N_accept = -1
  svr.V_accept = ""
  svr.Peers = nil

  return &svr
}

// Log server for debugging purposes.
func (svr *Server) info(fun string) {
  log.Printf("S%d %s:  T = %d  N_h = %d  N_a = %d  V_a = %s\n",
      svr.Id, fun, svr.Time, svr.N_high, svr.N_accept, svr.V_accept)
}

// Increment the server's Lamport clock. The low digit is the ID of the node,
// which serves to break ties.
func (svr *Server) update() {
  svr.Time += 10
}

// Initiate the Paxos protocol to append a new string to the log.
func (svr *Server) AppendLog(args *echoproto.Args, reply *echoproto.Reply) error {
  var pargs echoproto.Args
  var preply echoproto.Reply
  var n_prepare, n_accept, n_error int
  var err error

  n_prepare = 0
  n_accept = 0

  log.Println("AppendLog(): Initiating Paxos protocol.")

  for n_accept <= svr.F {
    for n_prepare <= svr.F {
      // initiate a commit to the log
      svr.Lock.Lock()

      // increment the Lamport timestamp and append the string to the log
      svr.update()
      pargs.N = svr.Time
      svr.info("AppendLog()")

      svr.Lock.Unlock()

      n_prepare = 0
      n_error = 0

      for i := 0; i < len(svr.Peers); i++ {
        err = svr.Peers[i].Call("Server.Prepare", pargs, &preply)

        if err != nil {
          log.Println("rpc.Call() error:", err)
          n_error++
        } else if preply.Response == echoproto.PREPARE_OK {
          n_prepare++
        }
      }

      if n_error > svr.F {
        log.Fatalln("Too many failed nodes. No progress possible.")
      }

      if n_prepare <= svr.F {
        log.Println("AppendLog(): Majority PREPARE_REJECT. Restarting Paxos.")
        time.Sleep(1000 * time.Millisecond)
      }
    }

    svr.Lock.Lock()

    svr.info("AppendLog()")
    log.Println("AppendLog(): Majority PREPARE_OK. Initiating accept phase.")

    // n_prepare > svr.F
    // continue with accept
    if svr.V_accept == "" {
      svr.V_accept = args.V
    }

    pargs.V = svr.V_accept

    svr.Lock.Unlock()

    n_accept = 0
    n_error = 0

    for i := 0; i < len(svr.Peers); i++ {
      err = svr.Peers[i].Call("Server.Accept", pargs, &preply)

      if err != nil {
        log.Println("rpc.Call() error:", err)
        n_error++
      } else if preply.Response == echoproto.ACCEPT_OK {
        n_accept++
      }
    }

    if n_error > svr.F {
      log.Fatalln("Too many failed nodes. No progress possible.")
    }

    if n_accept <= svr.F {
      log.Println("AppendLog(): Majority ACCEPT_REJECT. Restarting Paxos.")
      time.Sleep(1000 * time.Millisecond)
    }
  }

  // n_accept > svr.F
  // continue with commit
  log.Println("AppendLog(): Majority ACCEPT_OK. Initiating commit phase.")

  n_error = 0

  for i := 0; i < len(svr.Peers); i++ {
    err = svr.Peers[i].Call("Server.Commit", pargs, &preply)

    if err != nil {
      log.Println("rpc.Call() error:", err)
      n_error++
    }
  }

  if n_error > svr.F {
    log.Fatalln("Too many failed nodes. No progress possible.")
  }

  svr.Commit(args, reply)

  return nil
}

func (svr *Server) FetchLog(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()

  // increment the Lamport timestamp and copy the current version of the log
  svr.update()
  reply.Data = svr.Log

  svr.info("FetchLog()")

  svr.Lock.Unlock()

  return nil
}

func (svr *Server) Register(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()

  svr.Store = append(svr.Store, args.V)

  // increment the Lamport timestamp and copy the current version of the log
  svr.update()
  reply.Data = svr.Log

  svr.info("FetchLog()")

  svr.Lock.Unlock()

  return nil
}

// RPC interface for the Paxos protocol.
func (svr *Server) Prepare(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()

  if args.N >= svr.N_high {
    svr.N_high = args.N
    reply.Response = echoproto.PREPARE_OK
  } else {
    reply.Response = echoproto.PREPARE_REJECT
  }

  svr.info("Prepare()")

  svr.Lock.Unlock()

  return nil
}

func (svr *Server) Accept(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()

  if args.N >= svr.N_high {
    svr.N_high = args.N
    svr.N_accept = args.N
    svr.V_accept = args.V
    reply.Response = echoproto.ACCEPT_OK
  } else {
    reply.Response = echoproto.ACCEPT_REJECT
  }

  svr.info("Accept()")

  svr.Lock.Unlock()

  return nil
}

func (svr *Server) Commit(args *echoproto.Args, reply *echoproto.Reply) error {
  svr.Lock.Lock()

  svr.Log = append(svr.Log, args.V)

  // echo the response to the client
  reply.Data = nil
  reply.Data = append(reply.Data, args.V)

  svr.N_high = -1
  svr.N_accept = -1
  svr.V_accept = ""

  svr.info("Commit()")

  svr.Lock.Unlock()

  return nil
}

func connect(host string) *rpc.Client {
  cli, err := rpc.DialHTTP("tcp", host)

  for j := 0; (err != nil) && (j < echoproto.N_ATTEMPTS); j++ {
    log.Println("rpc.DialHTTP() failed, retrying")

    time.Sleep(1000 * time.Millisecond)
    cli, err = rpc.DialHTTP("tcp", host)
  }

  if err != nil {
    log.Fatalln("rpc.DialHTTP() error:", err.Error())
  }

  return cli
}

func main() {
  var svr *Server
  var clients []*rpc.Client
  var hostname string
  var block chan int
  var err error

  var ihelp *bool = flag.Bool("h", false, "Print help information")
  var iport *int = flag.Int("p", 55455, "Port number")
  var id *int = flag.Int("i", 1, "Lamport clock ID number")

  flag.Parse()
  if *ihelp {
    flag.Usage()
    os.Exit(0)
  }

  runtime.GOMAXPROCS(8)

  // register a server object to the RPC interface defined above
  svr = NewServer(*id, flag.NArg() / 2)
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

  clients = nil

  for i := 0; i < flag.NArg(); i++ {
    clients = append(clients, connect(flag.Arg(i)))
  }

  svr.Peers = clients
  svr.Lock.Unlock()

  block = make(chan int)
  <- block
}
