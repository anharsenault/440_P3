package echoproto

const (
  PREPARE_OK = iota
  PREPARE_REJECT
  ACCEPT_OK
  ACCEPT_REJECT
)

const(
  N_ATTEMPTS = 3
)

type Args struct {
  N int
  V string
}

type Reply struct {
  Data []string
  Response int
}
