package main

import (
	"flag"
	"fmt"
	"strings"
	"os"
	"P3-f12/official/lsp12"
	"P3-f12/official/lspnet"
	"P3-f12/official/lsplog"
)

func runserver(srv *lsp12.LspServer, pm *lsp12.LspClient) {

	for {
		// Read from client
		id, payload, rerr := srv.Read()
		if rerr != nil  {
			fmt.Printf("Connection %d has died.  Error message %s\n", id, rerr.Error())
		} else {
			s := string(payload)
			lsplog.Vlogf(6, "Connection %d.  Received '%s'\n", id, s)
			payload = []byte(strings.ToUpper(s))

/////////////// log to backup////////////////////////
    //pm.Write(payload)
/////////////////////////////////////////////////

			// Echo back to client
			werr := srv.Write(id, payload)
			if werr != nil {
				fmt.Printf("Connection %d.  Write failed.  Error message %s\n", id, werr.Error())
			}
		}
	}
}

func main() {
	var ihelp *bool = flag.Bool("h", false, "Print help information")
	var iport *int = flag.Int("p", 6666, "Port number")
	var iverb *int = flag.Int("v", 1, "Verbosity (0-6)")
	var idrop *int = flag.Int("r", 0, "Network packet drop percentage")
	var elim *int = flag.Int("k", 5, "Epoch limit")
	var ems *int = flag.Int("d", 2000, "Epoch duration (millisecconds)")
  var st_num *int = flag.Int("s", 2, "Number of storage server")

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

  var st_info map[string]uint16
  st_info = make(map[string]uint16)

  params := &lsp12.LspParams{*elim,*ems}
//////////////////////////////////////////////////////////
  //setup temporary lsp server to collect storage information
  st_port := 55455
  st, err := lsp12.NewLspServer(st_port, params)
  if err != nil {
		fmt.Printf("creat log server failed. Error message %s\n", err.Error())
	}

  for i := 0; i < *st_num; i++ {
    connID, st_host, _ := st.Read()
    fmt.Printf("receive [%s] from storage\n", st_host)
    st_info[string(st_host)] = connID
    st.CloseConn(connID)
  }

/////////////////////////////////////////////////////////
  var pm_st string
  var pm *lsp12.LspClient

  for key, _ := range st_info {
    pm_st = key
    pm, err = lsp12.NewLspClient(pm_st, params)
    if err != nil {
      fmt.Printf("pick up %s as pm failed.  Error message %s\n", pm_st, err.Error())
      //delete from st_info need handle here, try the rest 
    } else {
      break;
    }
  }
	fmt.Printf("Echo server establishing server on port %d\n", port)

  st_hosts := make([]string, len(st_info) - 1)
  i := 0
  for key, _ := range st_info {
    if key == pm_st {
      continue
    }
    st_hosts[i] = key
    i++
  }
  cmd := fmt.Sprintf("UPGRADE#%s",strings.Join(st_hosts, ";"))
  pm.Write([]byte(cmd))

  srv, err := lsp12.NewLspServer(*iport, params)
  if err != nil {
    fmt.Printf("fail to build up echo server on port %d\n", *iport)
  }

  runserver(srv, pm)
}
