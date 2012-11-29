package main

import (
  //"encoding/json"
  "strings"
	"flag"
	"fmt"
	"os"
	"P3-f12/official/lsp12"
	"P3-f12/official/lspnet"
	"P3-f12/official/lsplog"
)

var log []string

func append_log(str string) {
  log = append(log, str)
}

func fetch_log() []string {
  return log
}

func upgrade(st_peers string) [](*lsp12.LspClient){

  ss := strings.Split(string(st_peers), ";")

  fmt.Printf("peers len:%d\n", len(ss))

  if len(ss) != 0 {
    return ss
  }
  return nil
}

func runserver(srv *lsp12.LspServer) {
  //var st_peers [](*lsp12.LspClient)

	for {
		// Read from master
	  _, payload, rerr := srv.Read()
		if rerr != nil {
			fmt.Printf("Echo server died.  Error message %s\n", rerr.Error())
		} else {
      fmt.Printf("log storage server received '%s'\n", string(payload))
      ss := strings.Split(string(payload), "#")
      if ss == nil {
        fmt.Printf("parsing %s error\n", string(payload))
      }

      //fmt.Printf("cmd %s\n", ss[0])
      //fmt.Printf("param %s\n", ss[1])

      switch {
      case ss[0] == "UPGRADE":
        fmt.Printf("UPGRADE")
        upgrade(ss[1])
      case ss[0] == "APPEND":
        fmt.Printf("APPEND")
      case ss[0] == "FETCH":
        fmt.Printf("FETCH")
      }

      /*
      _, err := fmt.Sscanf(string(payload), "%d %s", &cmd, &str)
      if err != nil {
        fmt.Printf("parsing error:%s\n", err.Error())
      }

      switch {
      case cmd == APPEND:
        append_log(str)
      case cmd == FETCH:
        payload, err = json.Marshal(fetch_log())
        //srv.Write(payload)
      }
      */
		}
	}
}

func main() {
	var ihelp *bool = flag.Bool("h", false, "Print help information")
	var iport *int = flag.Int("p", 9000, "Port number")
  //var ihost *string = flag.String("H", "localhost", "Host address")
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

  params := &lsp12.LspParams{*elim,*ems}
  cli, err := lsp12.NewLspClient((*master), params)
  if err != nil {
		fmt.Printf("create lsp client failed. Error message %s\n", err.Error())
	}

  cli.Write([]byte(fmt.Sprintf("localhost:%d", *iport)))
  cli.Close()

  srv, err := lsp12.NewLspServer(*iport, params)
  if err != nil {
		fmt.Printf("setup server on port %d failed: %s\n", *iport, err.Error())
	}
  fmt.Printf("Establishing server on port %d\n", *iport)
  runserver(srv)

  /*
  hostport := fmt.Sprintf("%s:%v", *ihost, *iport)

	if err != nil {
		fmt.Printf("... failed.  Error message %s\n", err.Error())
	} else {
		runserver(cli)
	}
  */
}
