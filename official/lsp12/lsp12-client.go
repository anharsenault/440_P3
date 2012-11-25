package lsp12

import (
	"P3-f12/official/lsplog"
	"P3-f12/official/lspnet"
	"fmt"
	"time"
)

type LspMessageChan chan *LspMessage

////////////////////////////////////////////////////////////////////////////////
// Shared between server & client code
////////////////////////////////////////////////////////////////////////////////

// All information associated with single client
type lspConn struct {
	addr *lspnet.UDPAddr // Address of other end of connection
	connId  uint16  // Connection ID
	sendBuf *Buf   // Messages queued to send
	pendingMsg *LspMessage // Message that has been sent, but not yet ack'ed
	lastAck *LspMessage // Last ack sent
	nextSendSeqNum byte
	nextRecvSeqNum byte
	lastHeardEpoch int64
	// Have network operations stopped for this connection?
	stopNetworkFlag bool
	// Flags to support connection shutdown on server
	readDoneFlag  bool // Have all reads been completed
	writeDoneFlag bool // Have all writes been completed
}

func newConn(addr *lspnet.UDPAddr, connId uint16, epoch int64) *lspConn {
	con := new(lspConn)
	con.addr = addr
	con.connId = connId
	con.sendBuf = NewBuf()
	con.pendingMsg = nil
	con.lastAck = nil
	con.nextSendSeqNum = 0
	con.nextRecvSeqNum = 0
	con.lastHeardEpoch = epoch
	return con
}

// Return the Connection ID for a client
func (cli *LspClient) iConnId() uint16 {
	return cli.lspConn.connId
}

// Goroutine for triggering epoch events
func epochTrigger(ms int, ec chan int, stop *bool) {
	d := time.Duration(ms) * time.Millisecond
	for !*stop {
		time.Sleep(d)
		ec <- 1
	}
}

////////////////////////////////////////////////////////////////////////////////
// Client code
////////////////////////////////////////////////////////////////////////////////


type iLspClient struct {
	params *LspParams
	lspConn *lspConn
	udpConn *lspnet.UDPConn
	readBuf *Buf  // Results that are ready to be read
	appReadChan LspMessageChan   // Supply results for Creation & Read functions
	appWriteChan LspMessageChan  // Requests to write
	netInChan LspMessageChan // Inputs from network
	epochChan chan int  // For triggering epochs
	currentEpoch int64
	stopAppFlag bool
	// For communicating results back to function calls
	closeReplyChan chan error 
	writeReplyChan chan error
}

func iNewLspClient(hostport string, params *LspParams) (*LspClient, error) {
	cli := new(LspClient)
	if params == nil {
		// Insert default parameters
		params = &LspParams{5,2000}
	}
	cli.params = params
	addr, err := lspnet.ResolveUDPAddr("udp", hostport)
	if lsplog.CheckReport(1, err) {
		return nil, err
	}
	cli.lspConn = newConn(addr, 0, 0)
	// Client's first received message will be data message.
	cli.lspConn.nextRecvSeqNum = NextSeqNum(0)
	cli.udpConn, err = lspnet.DialUDP("udp", nil, addr)
	if lsplog.CheckReport(1, err) {
		return nil, err
	}
	// Need enough room to recycle close messages at end
	cli.appReadChan = make(LspMessageChan, 2)
	cli.readBuf = NewBuf()
	cli.appWriteChan = make(LspMessageChan, 1)
	cli.netInChan = make(LspMessageChan, 1)
	cli.epochChan = make(chan int)
	cli.closeReplyChan = make(chan error, 2)
	cli.writeReplyChan = make(chan error, 2)

	go cli.clientLoop()
	go cli.udpReader()
	go epochTrigger(cli.params.EpochMilliseconds, cli.epochChan, &cli.lspConn.stopNetworkFlag)
	// Send connection request to server
	nm := GenConnectMessage()
	cli.udpWrite(nm)
	cli.lspConn.pendingMsg = nm
	cli.lspConn.nextSendSeqNum = NextSeqNum(0)
	cm := <- cli.appReadChan
	if cm.Type == MsgCONNECT {
		return cli, nil
	}
	return nil, lsplog.MakeErr("Connection failed")
}

// Main client loop
func (cli *LspClient) clientLoop() {
	for !(cli.stopAppFlag && cli.lspConn.stopNetworkFlag) {
		if cli.readBuf.Empty() {
			select {
			case netm := <-cli.netInChan:
				cli.handleNetMessage(netm)
			case appm := <-cli.appWriteChan:
				cli.handleAppWrite(appm)
			case <- cli.epochChan:
				cli.handleEpoch()
			}
		} else {
			v := cli.readBuf.Front()
			rm := v.(*LspMessage)
			if rm.Type == MsgINVALID {
				// Have completed all reads.  Stop applications
				cli.stopAppFlag = true
			}
			select {
			case netm := <-cli.netInChan:
				cli.handleNetMessage(netm)
			case appm := <-cli.appWriteChan:
				cli.handleAppWrite(appm)
			case <- cli.epochChan:
				cli.handleEpoch()
			case cli.appReadChan <- rm:
				cli.readBuf.Remove()
			}
		}
		cli.checkToSend()
	}
	// Make sure any subsequent operations fail
	cli.closeReplyChan <- nil
	cli.writeReplyChan <- lsplog.ConnectionClosed()
	cm := GenInvalidMessage(0, 0)
	cli.appReadChan <- cm
}

// Receive message from network
func (cli *LspClient) handleNetMessage(netm *LspMessage) {
	lspConn := cli.lspConn
	lspConn.lastHeardEpoch = cli.currentEpoch
	switch netm.Type {
	case MsgDATA:
		if lspConn.connId == 0 {
			cli.Vlogf(6, "Data received when connection not yet established\n")
			return
		}
		n := lspConn.nextRecvSeqNum
		if netm.SeqNum == n {
			cli.readBuf.Insert(netm)
			lspConn.nextRecvSeqNum = NextSeqNum(n)
			// Generate acknowledgement
			lspConn.lastAck = GenAckMessage(lspConn.connId, n)
			cli.udpWrite(lspConn.lastAck)
			cli.Vlogf(4, "Received & acknowledged %s\n", netm)
		} else {
			cli.Vlogf(6, "Ignoring data message #%v.  Expecting %v\n",
				netm.SeqNum, n)
			return
		}
	case MsgACK:
		if lspConn.pendingMsg == nil {
			cli.Vlogf(6, "Ignoring ack message #%v.  No message pending\n",
				netm.SeqNum)
			return
		}
		n := lspConn.pendingMsg.SeqNum
		if netm.SeqNum == n {
			cli.Vlogf(5, "Acknowledgement %v received\n", n)
			if n == 0 {
				if lspConn.pendingMsg.Type == MsgCONNECT {
					lspConn.connId = netm.ConnId
					cli.Vlogf(3, "Connected to server with ID %v\n",
						netm.ConnId)
					// Set up acknowledgement message with sequence number 0
					// for epoch events
					lspConn.lastAck = GenAckMessage(lspConn.connId, 0)
					// Let NewLspClient know that connection is established
					cli.appReadChan <- lspConn.pendingMsg
				} else {
					cli.Vlogf(6, "Expected Connect message.  Got %s\n",
						typeName[lspConn.pendingMsg.Type])
				}
			}
			lspConn.pendingMsg = nil
		} else {
			cli.Vlogf(6, "Ignoring ack message #%v.  Expecting %v\n",
				netm.SeqNum, n)
		}
	default:
		cli.Vlogf(6, "Ignoring message of type %s\n", typeName[netm.Type])
		return
	}
}

// Process write or close
func (cli *LspClient) handleAppWrite(appm *LspMessage) {
	// Queue data or close message to send over network
	cli.lspConn.sendBuf.Insert(appm)
	if appm.Type == MsgINVALID {
		cli.stopApp(true)
	} else {
		cli.writeReplyChan <- nil
	}
}

// Process epoch event
func (cli *LspClient) handleEpoch() {
	cli.currentEpoch ++
	if int(cli.currentEpoch - cli.lspConn.lastHeardEpoch) > cli.params.EpochLimit {
		cli.Vlogf(3, "Epoch limit of %v exceeded.\n", cli.params.EpochLimit)
		// Shut down network & apps
		cli.stopNetwork()
		// Not ready to stop reads
		cli.stopApp(false)
		// See if have failed to get connection
		if cli.lspConn.connId == 0 {
			cli.Vlogf(5, "Failed to establish connection\n")
			// Send signal to NewLspClient
			cli.appReadChan <- GenInvalidMessage(0, 0)
		}
	} else {
		pm := cli.lspConn.pendingMsg
		if pm != nil {
			cli.Vlogf(6, "Resending message %s\n", pm)
			cli.udpWrite(pm)
		}
		am := cli.lspConn.lastAck
		if am != nil {
			cli.Vlogf(6, "Resending ack #%v\n", am.SeqNum)
			cli.udpWrite(am)
		}
	}
}

// See if we can send any messages
func (cli *LspClient) checkToSend() {
	con := cli.lspConn
	if !con.sendBuf.Empty() && con.connId > 0 && con.pendingMsg == nil {
		d := con.sendBuf.Remove()
		sm := d.(*LspMessage)
		n := con.nextSendSeqNum
		sm.ConnId = con.connId
		sm.SeqNum = n
		con.nextSendSeqNum = NextSeqNum(n)
		if sm.Type == MsgINVALID {
			// Have cleared out send buffer and can now close connection
			cli.Vlogf(6, "All messages sent.  Closing connection\n")
			cli.stopNetwork()
		} else {
			con.pendingMsg = sm
			cli.Vlogf(4, "Sending message %s\n", sm)
			cli.udpWrite(sm)
		}
	}
}

// Goroutine that reads messages from UDP connection and writes to message channel
func (cli *LspClient) udpReader() {
	udpConn := cli.udpConn
	mc := cli.netInChan
	var buffer [1500] byte
	for !cli.lspConn.stopNetworkFlag {
		n, _, err := udpConn.ReadFromUDP(buffer[0:])
		if lsplog.CheckReport(1, err) {
			cli.Vlogf(6, "Client continuing\n")
			continue
		}
		m, merr := extractMessage(buffer[0:n])
		if lsplog.CheckReport(1, merr) {
			cli.Vlogf(6, "Client continuing\n")
			continue
		}
		mc <- m
	}
}

// Write message to UDP connection.  Address already registered with connection
func (cli *LspClient) udpWrite(msg *LspMessage) {
	b := msg.genPacket()
	_, err := cli.udpConn.Write(b)
	if lsplog.CheckReport(6, err) {
		cli.Vlogf(6, "Write failed\n")
	}
}

// Reporting from client
func (cli *LspClient) Vlogf(level int, format string, v ...interface{}) {
	nformat := fmt.Sprintf("C%v: %s", cli.lspConn.connId, format)
	lsplog.Vlogf(level, nformat, v...)
}

// Shutting down network communications
func (cli *LspClient) stopNetwork() {
	cli.lspConn.stopNetworkFlag = true
	err := cli.udpConn.Close()
	if lsplog.CheckReport(4, err) {
		lsplog.Vlogf(6, "Client Continuing\n")
	}
}

// Shutting down app read/writes
func (cli *LspClient) stopApp(setFlag bool) {
	// Send close message to application
	cli.Vlogf(6, "Disabling reads\n")
	cm := GenInvalidMessage(0, 0)
	cli.readBuf.Insert(cm)
	if setFlag {
		cli.stopAppFlag = true
	}
}


func (cli *LspClient) iRead() ([]byte, error) {
	m := <- cli.appReadChan
	switch m.Type {
	case MsgDATA:
		return m.Payload, nil
	case MsgINVALID:
		// Indicates that read should fail
		// Recycle message so that future reads will also fail
		cli.appReadChan <- m
		return nil, lsplog.ConnectionClosed()
	}
	return nil, lsplog.ConnectionClosed()
}

func (cli *LspClient) iWrite(payload []byte) error {
	// Will fill in ID & sequence number later
	m := GenDataMessage(0, 0, payload)
	cli.appWriteChan <- m
	rm := <- cli.writeReplyChan
	lsplog.Vlogf(5, "Completed write of %s", string(payload))
	if rm != nil {
		// Recycle so that subsequent writes will get error
		cli.writeReplyChan <- rm
	}
	return rm
}

func (cli *LspClient) iClose() {
	m := GenInvalidMessage(0, 0)
	cli.appWriteChan <- m
	<- cli.closeReplyChan
	// Put back nil, so that subsequent closes will succeed
	cli.closeReplyChan <- nil
}

