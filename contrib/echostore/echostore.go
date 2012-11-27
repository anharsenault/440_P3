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

  werr := svr.Write(nil)
  payload, rerr := svr.Read()

	for {
		// Read from master
		id, payload, rerr := srv.Read()
		if rerr != nil {
			fmt.Printf("Connection %d has died.  Error message %s\n", id, rerr.Error())
		} else {
      _, err := fmt.Sscanf(string(payload), "%d %s", &cmd, &str)
      if lsplog.CheckReport(6, err) {
        continue;
      }

			lsplog.Vlogf(6, "Connection %d.  Received '%s'\n", id, string(payload))
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
	var iport *int = flag.Int("p", 0, "Port number")
  var imaster *int = flag.Int("master", "localhost:55455", "Master server")
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
	//srv, err := lsp12.NewLspServer(port, params)
  cli, err := lsp12.NewLspClient(*imaster, params)
	if err != nil {
		fmt.Printf("... failed.  Error message %s\n", err.Error())
	} else {
		runserver(cli)
	}
}
