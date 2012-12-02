package echoproto

const (
  PREPARE_OK = iota
  PREPARE_REJECT

)

type Args struct {
  Str string
  N int
}

type Reply struct {
  Data []string
  Answer int
}

type PrepareArgs struct {

}
