package main

import (
  "encoding/json"
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

const (
    APPEND = iota
    FETCH
)

func runserver(srv *lsp12.LspClient) {
  var str string
  var cmd int

  srv.Write(nil)
  //werr := srv.Write(nil)

	for {
		// Read from master
	  payload, rerr := srv.Read()
		if rerr != nil {
			fmt.Printf("Echo server died.  Error message %s\n", rerr.Error())
		} else {
      fmt.Printf("log storage server received '%s'\n", string(payload))

      _, err := fmt.Sscanf(string(payload), "%d %s", &cmd, &str)
      if err != nil {
        fmt.Printf("parsing error:%s\n", err.Error())
      }

      switch {
      case cmd == APPEND:
        append_log(str)
      case cmd == FETCH:
        payload, err = json.Marshal(fetch_log())
        srv.Write(payload)
      }
		}
	}
}

func main() {
	var ihelp *bool = flag.Bool("h", false, "Print help information")
	var iport *int = flag.Int("p", 55455, "Port number")
  var ihost *string = flag.String("H", "localhost", "Host address")
	var iverb *int = flag.Int("v", 1, "Verbosity (0-6)")
	var idrop *int = flag.Int("r", 0, "Network packet drop percentage")
	var elim *int = flag.Int("k", 5, "Epoch limit")
	var ems *int = flag.Int("d", 2000, "Epoch duration (millisecconds)")

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

	lsplog.SetVerbose(*iverb)
	lspnet.SetWriteDropPercent(*idrop)

	fmt.Printf("Establishing server on port %d\n", port)
  hostport := fmt.Sprintf("%s:%v", *ihost, *iport)

  cli, err := lsp12.NewLspClient(hostport, params)

	if err != nil {
		fmt.Printf("... failed.  Error message %s\n", err.Error())
	} else {
		runserver(cli)
	}
}
