package echoclient

import (
  "fmt"
  "net/rpc"
  "P3-f12/contrib/echoproto"
)

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
