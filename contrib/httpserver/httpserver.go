package main

import (
    "fmt"
    "net/http"
    "strings"
    "strconv"
    "P3-f12/contrib/echoclient"
    //"html"
    //"io/ioutil"
)

func handler(w http.ResponseWriter, r *http.Request) {
    //fmt.Printf("invoke !\n")

    reqFile := strings.Replace(fmt.Sprintf("%q", r.URL), "\"", "", -1)
    //fmt.Printf("after:[%s]\n", reqFile)

    // what a hack here
    if  reqFile == "/" {
      http.ServeFile(w, r, "index.html")
    } else {
      reqFile = fmt.Sprintf(".%s", reqFile)
      //fmt.Printf("request:[%s]\n", reqFile)
      http.ServeFile(w, r, reqFile)
    }
}

type HttpClient struct {
  bindAddr string
  echoClients [](*(echoclient.EchoClient))
}

func newHttpClient() (*HttpClient) {
  var cli HttpClient

  cli.bindAddr = ":55555"
  cli.echoClients = make([](*(echoclient.EchoClient)), 0)

  return &cli
}

func (hc *HttpClient)send(w http.ResponseWriter, r *http.Request) {
  idx, _ := strconv.Atoi(r.FormValue("index"))
  str := r.FormValue("para")

  response, err := hc.echoClients[idx].Send(str)
  if err != nil {
    fmt.Fprintf(w, "%s", err.Error())
    return
  }

  fmt.Fprintf(w, "%s", response)
}

func (hc *HttpClient)fetch(w http.ResponseWriter, r *http.Request) {
  idx, _ := strconv.Atoi(r.FormValue("index"))

  response, err := hc.echoClients[idx].Fetch()
  if err != nil {
    fmt.Fprintf(w, "%s", err.Error())
    return
  }

  fmt.Fprintf(w, "%s", response)
}

func (hc *HttpClient)newClient(w http.ResponseWriter, r *http.Request) {
  var ret = 0

  host := r.FormValue("host")

  cli, err := echoclient.NewEchoClient(host)
  if err != nil {
    ret = 1
    fmt.Fprintf(w, "%d", ret)
    return
  }

  hc.echoClients = append(hc.echoClients, cli)
  fmt.Fprintf(w, "%d", ret)
}

func (hc *HttpClient) init() {
  http.HandleFunc("/", handler)

  http.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request){
    hc.newClient(w, r)
  })

  http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request){
    hc.send(w, r)
  })

  http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request){
    hc.fetch(w, r)
  })
  http.ListenAndServe(hc.bindAddr, nil)
}

func main() {
  hcli := newHttpClient()
  hcli.init()
}
