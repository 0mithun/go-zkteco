package zkteco

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

// ZKTeco is the main client for connecting to ZKTeco devices.
type ZKTeco struct {
	host     string
	port     int
	protocol string
	timeout  time.Duration
	password int

	// TCPMUX proxy support
	tcpmuxEnabled   bool
	tcpmuxHost      string
	tcpmuxPort      int
	tcpmuxSubdomain string

	conn      net.Conn
	sessionID uint16
	replyID   uint16
	lastData  []byte
	tcpBuffer []byte
}

// Option configures a ZKTeco client.
type Option func(*ZKTeco)

// WithProtocol sets the protocol ("tcp" or "udp"). Default is "udp".
func WithProtocol(protocol string) Option {
	return func(z *ZKTeco) {
		z.protocol = strings.ToLower(protocol)
	}
}

// WithTimeout sets the socket timeout in seconds. Default is 25.
func WithTimeout(seconds int) Option {
	return func(z *ZKTeco) {
		z.timeout = time.Duration(seconds) * time.Second
	}
}

// WithPassword sets the device password. Default is 0 (no password).
func WithPassword(password int) Option {
	return func(z *ZKTeco) {
		z.password = password
	}
}

// WithTCPMUX enables TCPMUX proxy support.
// host is the TCPMUX proxy host, port is the TCPMUX proxy port,
// subdomain is used to build the HTTP CONNECT target.
func WithTCPMUX(host string, port int, subdomain string) Option {
	return func(z *ZKTeco) {
		z.tcpmuxEnabled = true
		z.tcpmuxHost = host
		z.tcpmuxPort = port
		z.tcpmuxSubdomain = subdomain
		z.protocol = "tcp" // TCPMUX always uses TCP
	}
}

// NewZKTeco creates a new ZKTeco client.
func NewZKTeco(host string, port int, opts ...Option) *ZKTeco {
	z := &ZKTeco{
		host:     host,
		port:     port,
		protocol: "udp",
		timeout:  25 * time.Second,
		password: 0,
		replyID:  65534,
	}
	for _, opt := range opts {
		opt(z)
	}
	return z
}

// IsTCP returns true if using TCP protocol.
func (z *ZKTeco) IsTCP() bool {
	return z.protocol == "tcp"
}

// Connect establishes a connection to the ZKTeco device.
func (z *ZKTeco) Connect() error {
	var err error

	if z.tcpmuxEnabled {
		// TCPMUX: connect to proxy, then HTTP CONNECT handshake
		proxyAddr := fmt.Sprintf("%s:%d", z.tcpmuxHost, z.tcpmuxPort)
		z.conn, err = net.DialTimeout("tcp", proxyAddr, z.timeout)
		if err != nil {
			return fmt.Errorf("dial tcpmux proxy %s: %w", proxyAddr, err)
		}

		if err := z.httpConnectHandshake(); err != nil {
			z.conn.Close()
			z.conn = nil
			return fmt.Errorf("tcpmux handshake: %w", err)
		}
	} else {
		addr := fmt.Sprintf("%s:%d", z.host, z.port)
		if z.IsTCP() {
			z.conn, err = net.DialTimeout("tcp", addr, z.timeout)
		} else {
			z.conn, err = net.DialTimeout("udp", addr, z.timeout)
		}
		if err != nil {
			return fmt.Errorf("dial %s %s: %w", z.protocol, addr, err)
		}
	}

	z.sessionID = 0
	z.replyID = 65534
	z.tcpBuffer = nil

	resp, err := z.command(CMD_CONNECT, nil, "general")
	if err != nil {
		z.conn.Close()
		return fmt.Errorf("connect command: %w", err)
	}

	pkt, err := parsePacket(resp)
	if err != nil {
		z.conn.Close()
		return fmt.Errorf("parse connect response: %w", err)
	}

	z.sessionID = pkt.SessionID

	if pkt.Command == CMD_ACK_UNAUTH {
		authKey := makeCommKey(z.password, z.sessionID)
		resp2, err := z.command(CMD_ACK_AUTH, authKey, "general")
		if err != nil {
			z.conn.Close()
			return fmt.Errorf("auth command: %w", err)
		}
		pkt2, err := parsePacket(resp2)
		if err != nil {
			z.conn.Close()
			return fmt.Errorf("parse auth response: %w", err)
		}
		if pkt2.Command != CMD_ACK_OK {
			z.conn.Close()
			return fmt.Errorf("authentication failed: command=%d", pkt2.Command)
		}
	}

	return nil
}

// Disconnect closes the connection.
func (z *ZKTeco) Disconnect() error {
	if z.conn == nil {
		return nil
	}
	z.command(CMD_EXIT, nil, "general")
	z.sessionID = 0
	err := z.conn.Close()
	z.conn = nil
	return err
}

// httpConnectHandshake performs HTTP CONNECT through a TCPMUX proxy.
func (z *ZKTeco) httpConnectHandshake() error {
	target := fmt.Sprintf("%s.%s:%d", z.tcpmuxSubdomain, z.host, z.port)

	request := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: Keep-Alive\r\n\r\n", target, target)

	z.conn.SetWriteDeadline(time.Now().Add(z.timeout))
	if _, err := z.conn.Write([]byte(request)); err != nil {
		return fmt.Errorf("send CONNECT request: %w", err)
	}

	z.conn.SetReadDeadline(time.Now().Add(z.timeout))
	reader := bufio.NewReader(z.conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read proxy response: %w", err)
	}

	// Read remaining headers until blank line
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read proxy headers: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	// Check for HTTP 200
	statusLine = strings.TrimSpace(statusLine)
	if !strings.Contains(statusLine, " 200 ") {
		return fmt.Errorf("proxy returned: %s", statusLine)
	}

	return nil
}

// command sends a command and receives the response.
func (z *ZKTeco) command(cmd uint16, data []byte, cmdType string) ([]byte, error) {
	if len(z.lastData) >= 8 {
		z.replyID = binary.LittleEndian.Uint16(z.lastData[6:8])
	}

	pkt, nextReplyID := createHeader(cmd, z.sessionID, z.replyID, data)

	if err := z.sendData(pkt); err != nil {
		return nil, err
	}

	resp, err := z.recvData()
	if err != nil {
		return nil, err
	}

	z.replyID = nextReplyID
	z.lastData = resp

	if cmdType == "data" {
		return resp, nil
	}

	if z.sessionID != 0 && len(resp) >= 6 {
		respSessionID := binary.LittleEndian.Uint16(resp[4:6])
		if respSessionID != z.sessionID {
			return nil, fmt.Errorf("session mismatch: expected %d got %d", z.sessionID, respSessionID)
		}
	}

	return resp, nil
}

// sendData sends raw packet data, wrapping with TCP header if needed.
func (z *ZKTeco) sendData(data []byte) error {
	if z.conn == nil {
		return fmt.Errorf("not connected")
	}

	z.conn.SetWriteDeadline(time.Now().Add(z.timeout))

	var toSend []byte
	if z.IsTCP() {
		toSend = wrapTCP(data)
	} else {
		toSend = data
	}

	_, err := z.conn.Write(toSend)
	return err
}

// recvData receives a response, handling TCP framing if needed.
func (z *ZKTeco) recvData() ([]byte, error) {
	if z.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	z.conn.SetReadDeadline(time.Now().Add(z.timeout))

	if z.IsTCP() {
		return z.recvTCP()
	}
	return z.recvUDP()
}

// recvUDP receives a single UDP packet.
func (z *ZKTeco) recvUDP() ([]byte, error) {
	buf := make([]byte, 65536)
	n, err := z.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	result := make([]byte, n)
	copy(result, buf[:n])
	return result, nil
}

// recvTCP receives a complete TCP-framed packet, handling buffering.
func (z *ZKTeco) recvTCP() ([]byte, error) {
	for {
		if payload, remainder, ok := extractTCPPacket(z.tcpBuffer); ok {
			z.tcpBuffer = remainder
			return payload, nil
		}

		buf := make([]byte, 16384)
		n, err := z.conn.Read(buf)
		if err != nil {
			return nil, err
		}
		z.tcpBuffer = append(z.tcpBuffer, buf[:n]...)
	}
}

// extractTCPPacket tries to extract a complete TCP-framed packet from buffer.
func extractTCPPacket(buf []byte) ([]byte, []byte, bool) {
	if len(buf) < 8 {
		return nil, buf, false
	}

	if buf[0] != 0x50 || buf[1] != 0x50 || buf[2] != 0x82 || buf[3] != 0x7D {
		return nil, buf, false
	}

	payloadLen := int(binary.LittleEndian.Uint32(buf[4:8]))
	totalLen := 8 + payloadLen

	if len(buf) < totalLen {
		return nil, buf, false
	}

	payload := make([]byte, payloadLen)
	copy(payload, buf[8:totalLen])

	var remainder []byte
	if len(buf) > totalLen {
		remainder = make([]byte, len(buf)-totalLen)
		copy(remainder, buf[totalLen:])
	}

	return payload, remainder, true
}

// recvLargeData receives chunked large data after CMD_PREPARE_DATA.
func (z *ZKTeco) recvLargeData(prepareResp []byte) ([]byte, error) {
	if len(prepareResp) < 12 {
		return nil, fmt.Errorf("PREPARE_DATA response too short: %d bytes", len(prepareResp))
	}

	totalSize := int(binary.LittleEndian.Uint32(prepareResp[8:12]))
	if totalSize <= 0 {
		return nil, nil
	}

	var allData []byte
	received := 0
	first := true

	for received < totalSize {
		var chunk []byte
		var err error

		if z.IsTCP() {
			chunk, err = z.readNextTCPPayload()
		} else {
			buf := make([]byte, 65536)
			z.conn.SetReadDeadline(time.Now().Add(z.timeout))
			n, readErr := z.conn.Read(buf)
			if readErr != nil {
				err = readErr
			} else {
				chunk = buf[:n]
			}
		}

		if err != nil {
			return nil, fmt.Errorf("receive chunk: %w", err)
		}

		if first {
			allData = append(allData, chunk...)
			if len(chunk) > 8 {
				received += len(chunk) - 8
			}
			first = false
		} else {
			if len(chunk) > 8 {
				allData = append(allData, chunk[8:]...)
				received += len(chunk) - 8
			} else {
				allData = append(allData, chunk...)
				received += len(chunk)
			}
		}
	}

	// Consume final ACK
	finalResp, err := z.recvData()
	if err != nil {
		return nil, fmt.Errorf("receive final ACK: %w", err)
	}
	z.lastData = finalResp

	return allData, nil
}

// readNextTCPPayload reads the next complete TCP-framed payload
func (z *ZKTeco) readNextTCPPayload() ([]byte, error) {
	for attempts := 0; attempts < 50; attempts++ {
		if payload, remainder, ok := extractTCPPacket(z.tcpBuffer); ok {
			z.tcpBuffer = remainder
			return payload, nil
		}

		buf := make([]byte, 16384)
		z.conn.SetReadDeadline(time.Now().Add(z.timeout))
		n, err := z.conn.Read(buf)
		if err != nil {
			return nil, err
		}
		z.tcpBuffer = append(z.tcpBuffer, buf[:n]...)
	}
	return nil, fmt.Errorf("readNextTCPPayload: exceeded max attempts")
}

// commandData sends a command expecting a large data response.
func (z *ZKTeco) commandData(cmd uint16, data []byte) ([]byte, error) {
	resp, err := z.command(cmd, data, "data")
	if err != nil {
		return nil, err
	}

	pkt, err := parsePacket(resp)
	if err != nil {
		return nil, err
	}

	if pkt.Command == CMD_PREPARE_DATA {
		return z.recvLargeData(resp)
	}

	if pkt.Command == CMD_ACK_DATA || pkt.Command == CMD_ACK_OK {
		return resp, nil
	}

	return nil, fmt.Errorf("unexpected response command: %d", pkt.Command)
}
