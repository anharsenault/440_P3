package lsp12

import (
	"P3-f12/official/lsplog"
	"P3-f12/official/lspnet"
	"fmt"
)

// Input stream from network must include source address
type networkData struct {
	msg *LspMessage
	addr *lspnet.UDPAddr
}

type networkChan chan *networkData

type iLspServer struct {
	nextId uint16
	params *LspParams
	udpConn *lspnet.UDPConn
	readBuf *Buf  // Results that are ready to be read
	appReadChan LspMessageChan   // Supply results for Read function
	appWriteChan LspMessageChan  // Requests to write or close
	netInChan networkChan // Inputs from network
	epochChan chan int  // For triggering epochs
	currentEpoch int64
	connById map[uint16] *lspConn  // All active connections, indexed by connId
	connByAddr map[string] *lspConn // All active connections, indexed by hostport
	stopGlobalNetworkFlag bool // All network operations terminate
	stopAppFlag bool
	// For communicating results back to function calls
	closeReplyChan chan error
	closeAllReplyChan chan error
	writeReplyChan chan error
}

func iNewLspServer(port int, params *LspParams) (*LspServer, error) {
	srv := new(LspServer)
	srv.nextId = 1
	if params == nil {
		// Insert default parameters
		params = &LspParams{5,2000}
	}
	srv.params = params
	hostport := fmt.Sprintf(":%v", port)
	addr, err := lspnet.ResolveUDPAddr("udp", hostport)
	if lsplog.CheckReport(1, err) {
		return nil, err
	}
	srv.udpConn, err = lspnet.ListenUDP("udp", addr)
	if lsplog.CheckReport(1, err) {
		return nil, err
	}
	srv.readBuf = NewBuf()
	// Need enough room to recycle close messages
	srv.appReadChan = make(LspMessageChan, 1)
	srv.appWriteChan = make(LspMessageChan)
	srv.netInChan = make(networkChan)
	srv.epochChan = make(chan int)
	srv.connById = make(map[uint16] *lspConn)
	srv.connByAddr = make(map[string] *lspConn)
	srv.closeReplyChan = make(chan error, 1)
	srv.closeAllReplyChan = make(chan error, 1)
	srv.writeReplyChan = make(chan error, 1)

	go srv.serverLoop()
	go srv.udpReader()
	go epochTrigger(srv.params.EpochMilliseconds, srv.epochChan, &srv.stopGlobalNetworkFlag)
	return srv, nil
}

// Main server loop
func (srv *LspServer) serverLoop() {
	for !(srv.stopAppFlag && srv.stopGlobalNetworkFlag) {
		var id uint16 = 0
		// Filter out any invalid messages from front of read buffer
		srv.filterReadBuf()
		if srv.readBuf.Empty() {
			select {
			case netd := <-srv.netInChan:
				id = srv.handleNetMessage(netd)
			case appm := <-srv.appWriteChan:
				id = srv.handleAppWrite(appm)
			case <- srv.epochChan:
				srv.handleEpoch()
			}
		} else {
			v := srv.readBuf.Front()
			rm := v.(*LspMessage)
			select {
			case netm := <-srv.netInChan:
				id = srv.handleNetMessage(netm)
			case appm := <-srv.appWriteChan:
				id = srv.handleAppWrite(appm)
			case <- srv.epochChan:
				srv.handleEpoch()
			case srv.appReadChan <- rm:
				srv.readBuf.Remove()
			}
		}
		srv.checkToSend(id)
	}
	srv.closeAllReplyChan <- nil
}

// Receive message from network.  If status changes for connection, return id
func (srv *LspServer) handleNetMessage(netd *networkData) uint16 {
	netm := netd.msg
	id := netm.ConnId
	con := srv.connById[id]
	if con == nil {
		if netm.Type != MsgCONNECT {
			srv.Vlogf(6, "Message with invalid Id %v received\n",
				netm.ConnId)
			return 0
		} 
	} else {
		con.lastHeardEpoch = srv.currentEpoch
	}
	switch netm.Type {
	case MsgCONNECT:
		if netm.SeqNum != 0 {
			srv.Vlogf(6, "Connection request with invalid sequence number %v\n",
				netm.SeqNum)
			return 0
		}
		// See if already have connection with this address:
		addr := netd.addr
		saddr := addr.String()
		ccon := srv.connByAddr[saddr] 
		if ccon != nil {
			srv.Vlogf(5, "Duplicate connection request from %s.  Resending Ack\n",
				saddr)
			// Resend acknowledgement
			srv.udpWrite(ccon, ccon.lastAck)
			ccon.lastHeardEpoch = srv.currentEpoch
			return 0
		}
		// New connection
		id = srv.nextId
		srv.nextId++
		con := newConn(addr, id, srv.currentEpoch)
		srv.connById[id] = con
		srv.connByAddr[saddr] = con
		// Data messages start with seqnum 1
		con.nextSendSeqNum = NextSeqNum(0)
		con.nextRecvSeqNum = NextSeqNum(0)
		srv.Vlogf(3, "Opening connection %d to %s\n", id, saddr)
		// Send acknowledgement
		con.lastAck = GenAckMessage(id, 0)
		srv.udpWrite(con, con.lastAck)
		return id
	case MsgDATA:
		n := con.nextRecvSeqNum
		if con.readDoneFlag {
			srv.Vlogf(6, "Ignoring data message on %v.  Connection closed\n", con.connId)
		} else if netm.SeqNum == n {
			srv.readBuf.Insert(netm)
			con.nextRecvSeqNum = NextSeqNum(n)
			// Generate acknowledgement
			con.lastAck = GenAckMessage(con.connId, n)
			srv.udpWrite(con, con.lastAck)
			srv.Vlogf(5, "Received & acknowledged %s\n", netm)
		} else {
			srv.Vlogf(6, "Ignoring data message #%v on %v.  Expecting %v\n",
				netm.SeqNum, con.connId, n)
			return 0 // Will not enable new send
		}
	case MsgACK:
		if con.pendingMsg == nil {
			srv.Vlogf(6, "Ignoring ack message #%v on %v.  No pending message\n",
				netm.SeqNum, con.connId)
			return 0
		}
		n := con.pendingMsg.SeqNum
		if netm.SeqNum == n {
			srv.Vlogf(5, "Acknowledement %v received on connection %v\n",
				n, id)
			con.pendingMsg = nil
			return id
		} else {
			srv.Vlogf(6, "Ignoring ack message #%v.  Expecting %v\n",
				netm.SeqNum, n)
			return 0
		}
	default:
		srv.Vlogf(6, "Ignoring message of type %s\n", typeName[netm.Type])
		return 0
	}
	return 0
}

// Process write or close
func (srv *LspServer) handleAppWrite(appm *LspMessage) uint16 {
	id := appm.ConnId
	con := srv.connById[id]
	if con == nil {
		if id == 0 && appm.Type == MsgINVALID {
			srv.Vlogf(1, "Application requesting shutdown of server\n")
			for _, con := range srv.connById {
				// Initiate closing of this connection
				srv.readDone(con)
			}
			srv.stopApp()
		} else if id != 0 && appm.Type == MsgINVALID {
			// Call to close on already closed connection.
			srv.Vlogf(6, "Application called close on nonexistent (possibly closed) connection.\n")
// Not needed for nonblocking close
//			srv.closeReplyChan <- nil
		} else {
			srv.Vlogf(6, "Message %s has invalid connection Id\n", appm, id)
		}
		return 0
	}
	switch appm.Type {
	case  MsgDATA:
		// Queue message to send over network
		con.sendBuf.Insert(appm)
		srv.writeReplyChan <- nil
	case MsgINVALID:
		// Initiate closing of this connection
		srv.readDone(con)
	default:
		// Shouldn't happen
		srv.Vlogf(6, "Unexpected message type %s from app write\n",
		typeName[appm.Type])
	}
	return id
}

// Process epoch event
func (srv *LspServer) handleEpoch() {
	srv.currentEpoch ++
	for _, con := range srv.connById {
		if con.writeDoneFlag {
			continue
		}
		if int(srv.currentEpoch - con.lastHeardEpoch) > srv.params.EpochLimit {
			srv.Vlogf(3, "Epoch limit of %v exceeded on connection %v.\n",
				srv.params.EpochLimit, con.connId)
			srv.writeDone(con)
		} else {
			pm := con.pendingMsg
			if pm != nil {
				srv.Vlogf(6, "Resending message %s\n", pm)
				srv.udpWrite(con, pm)
			}
			am := con.lastAck
			if am != nil {
				srv.Vlogf(6, "Resending ack #%v on connection %v\n",
					am.SeqNum, am.ConnId)
				srv.udpWrite(con, am)
			}
		}
	}
	// See if it's time to shut down entire network
	if srv.stopAppFlag && len(srv.connById) == 0 {
		srv.stopGlobalNetwork()
	}
}


// See if we can send any messages for given Id
func (srv *LspServer) checkToSend(id uint16) {
	if id == 0 { return }
	con := srv.connById[id]
	if con == nil {
		srv.Vlogf(6, "Unexpected Id %v for checkToSend\n", id)
		return
	} 
	if !con.sendBuf.Empty() && con.pendingMsg == nil {
		d := con.sendBuf.Remove()
		sm := d.(*LspMessage)
		n := con.nextSendSeqNum
		con.nextSendSeqNum = NextSeqNum(n)
		sm.ConnId = con.connId
		sm.SeqNum = n
		if sm.Type == MsgINVALID {
			// Have cleared out send buffer
			srv.writeDone(con)
		} else {
			con.pendingMsg = sm
			srv.Vlogf(6, "Sending message %s\n", sm)
			srv.udpWrite(con, sm)
		}
	}
}

// Filter out any invalid messages from front of read buffer
func (srv *LspServer) filterReadBuf() {
	for !srv.readBuf.Empty() {
		v := srv.readBuf.Front()
		rm := v.(*LspMessage)
		id := rm.ConnId
		con := srv.connById[id]
		if rm.Type == MsgINVALID {
			// Keep message there
			return
		}
		if con == nil || con.readDoneFlag == true {
			srv.readBuf.Remove()
			srv.Vlogf(6, "Filtering out received message for closed connection '%s'\n", rm)
		} else {
			break
		}
	}
}



// Write message to UDP connection.  Address specified by con
func (srv *LspServer) udpWrite(con *lspConn, msg *LspMessage) {
	b := msg.genPacket()
	_, err := srv.udpConn.WriteToUDP(b, con.addr)
	if lsplog.CheckReport(6, err) {
		srv.Vlogf(6, "Write failed\n")
	}
}


// Goroutine that reads messages from UDP connection and writes to message channel
func (srv *LspServer) udpReader() {
	udpConn := srv.udpConn
	netc := srv.netInChan
	var buffer [1500] byte
	for !srv.stopGlobalNetworkFlag {
		n, addr, err := udpConn.ReadFromUDP(buffer[0:])
		if lsplog.CheckReport(1, err) {
			srv.Vlogf(5, "Server continuing\n")
			continue
		}
		m, merr := extractMessage(buffer[0:n])
		if lsplog.CheckReport(1, merr) {
			srv.Vlogf(6, "Server continuing\n")
			continue
		}
		srv.Vlogf(5, "Received message %s\n", m)
		d := &networkData{m,addr}
		netc <- d
	}
}

// Reporting from server
func (srv *LspServer) Vlogf(level int, format string, v ...interface{}) {
	nformat := fmt.Sprintf("S: %s", format)
	lsplog.Vlogf(level, nformat, v...)
}

// Shut down all network activity
func(srv *LspServer) stopGlobalNetwork() {
	srv.stopGlobalNetworkFlag = true
	err := srv.udpConn.Close()
	if lsplog.CheckReport(4, err) {
		lsplog.Vlogf(6, "Server Continuing\n")
	}
}

// Mark that have completed all reads for connection
func (srv *LspServer) readDone(con *lspConn) {
	srv.Vlogf(6, "Reads done for connection %v\n", con.connId)
	con.readDoneFlag = true
	if con.writeDoneFlag || (con.pendingMsg == nil && con.sendBuf.Empty()) {
		srv.deleteConnection(con)
	} else {
		// Insert message into send buffer to detect when writes are done
		m := GenInvalidMessage(con.connId, 0)
		con.sendBuf.Insert(m)
	}
}

// Mark that have completed all writes for connection
func (srv *LspServer) writeDone(con *lspConn) {
	srv.Vlogf(6, "Writes done for connection %v\n", con.connId)
	con.writeDoneFlag = true
	if con.readDoneFlag {
		srv.deleteConnection(con)
	} else {
		// Insert message into read buffer to detect when read done
		m := GenInvalidMessage(con.connId, 0)
		srv.readBuf.Insert(m)
		// Disable sending or resending any more messages
		con.pendingMsg = nil
	}
}

// Delete connection
func (srv *LspServer) deleteConnection(con *lspConn) {
	srv.Vlogf(6, "Deleting connection %v\n", con.connId)
	delete(srv.connById, con.connId)
	delete(srv.connByAddr, con.addr.String())
}

// Shut down app activity
func(srv *LspServer) stopApp() {
	// Send close message to application
	cm := GenInvalidMessage(0, 0)
	srv.readBuf.Insert(cm)
	srv.stopAppFlag = true
}


func (srv *LspServer) iRead() (uint16, []byte, error) {
	m := <- srv.appReadChan
	switch m.Type {
	case MsgDATA:
		return m.ConnId, m.Payload, nil
	case MsgINVALID:
		// Indicates that read should fail.
		return m.ConnId, nil, lsplog.ConnectionClosed()
	}
	return m.ConnId, nil, lsplog.ConnectionClosed()
}

func (srv *LspServer) iWrite(connId uint16, payload []byte) error {
	m := GenDataMessage(connId, 0, payload)
	srv.appWriteChan <- m
	rm := <- srv.writeReplyChan
	return rm
}

func (srv *LspServer) iCloseConn(connId uint16) {
	if connId == 0 {
		return
	}
	m := GenInvalidMessage(connId, 0)
	srv.appWriteChan <- m
// Not needed for nonblocking close
//	<- srv.closeReplyChan
}

// Close all connections and terminate server
func (srv *LspServer) iCloseAll() {
	// Notify server that want to close all connections
	m := GenInvalidMessage(0, 0)
	srv.appWriteChan <- m
	done := false
	// Skip over any connection closed replies
	for !done {
		select {
		case <- srv.closeReplyChan:
		case <-srv.closeAllReplyChan:
			done = true
		}
	}
	// Insert nil for subsequent closes
	srv.closeAllReplyChan <- nil
}
