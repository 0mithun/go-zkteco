package zkteco

import (
	"encoding/binary"
	"fmt"
	"time"
)

const ushrtMax = 65535

// tcpMagic is the TCP framing header
var tcpMagic = []byte{0x50, 0x50, 0x82, 0x7D}

// packet represents a parsed ZKTeco protocol packet
type packet struct {
	Command   uint16
	Checksum  uint16
	SessionID uint16
	ReplyID   uint16
	Data      []byte
}

// parsePacket parses raw bytes (without TCP framing) into a packet
func parsePacket(data []byte) (*packet, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("packet too short: %d bytes", len(data))
	}
	p := &packet{
		Command:   binary.LittleEndian.Uint16(data[0:2]),
		Checksum:  binary.LittleEndian.Uint16(data[2:4]),
		SessionID: binary.LittleEndian.Uint16(data[4:6]),
		ReplyID:   binary.LittleEndian.Uint16(data[6:8]),
	}
	if len(data) > 8 {
		p.Data = make([]byte, len(data)-8)
		copy(p.Data, data[8:])
	}
	return p, nil
}

// createHeader builds a ZKTeco packet with proper checksum.
// Returns the full packet bytes and the next replyID.
// Note: checksum is calculated with the original replyID, but the packet
// is sent with the incremented replyID (matching PHP behavior).
func createHeader(command uint16, sessionID uint16, replyID uint16, data []byte) ([]byte, uint16) {
	packetLen := 8 + len(data)
	buf := make([]byte, packetLen)

	// Step 1: Pack with original replyID and checksum=0 for checksum calculation
	binary.LittleEndian.PutUint16(buf[0:2], command)
	binary.LittleEndian.PutUint16(buf[2:4], 0)
	binary.LittleEndian.PutUint16(buf[4:6], sessionID)
	binary.LittleEndian.PutUint16(buf[6:8], replyID)
	if len(data) > 0 {
		copy(buf[8:], data)
	}

	// Step 2: Calculate checksum over original packet
	checksum := calculateChecksum(buf)

	// Step 3: Increment replyID (wrapping at USHRT_MAX)
	nextReplyID := replyID + 1
	if nextReplyID >= ushrtMax {
		nextReplyID -= ushrtMax
	}

	// Step 4: Repack with computed checksum and incremented replyID
	binary.LittleEndian.PutUint16(buf[2:4], checksum)
	binary.LittleEndian.PutUint16(buf[6:8], nextReplyID)

	return buf, nextReplyID
}

// calculateChecksum computes the ZKTeco 16-bit checksum
func calculateChecksum(data []byte) uint16 {
	var chksum int64 = 0
	length := len(data)

	for i := 0; i < length-1; i += 2 {
		val := int64(binary.LittleEndian.Uint16(data[i : i+2]))
		chksum += val
		if chksum > ushrtMax {
			chksum -= ushrtMax
		}
	}

	if length%2 != 0 {
		chksum += int64(data[length-1])
	}

	for chksum > ushrtMax {
		chksum -= ushrtMax
	}

	if chksum > 0 {
		chksum = -chksum
	}
	chksum--
	for chksum < 0 {
		chksum += ushrtMax
	}

	return uint16(chksum)
}

// wrapTCP wraps a packet with TCP framing header
func wrapTCP(packet []byte) []byte {
	result := make([]byte, 8+len(packet))
	copy(result[0:4], tcpMagic)
	binary.LittleEndian.PutUint32(result[4:8], uint32(len(packet)))
	copy(result[8:], packet)
	return result
}

// makeCommKey generates the auth key for password authentication.
// Matches PHP's makeCommKey exactly: reverses bits, XORs with "ZKSO", swaps halves.
func makeCommKey(key int, sessionID uint16) []byte {
	// Step 1: Reverse all 32 bits of the key
	var k uint32
	for i := 0; i < 32; i++ {
		if key&(1<<i) != 0 {
			k = (k << 1) | 1
		} else {
			k = k << 1
		}
	}

	// Step 2: Add session ID
	k += uint32(sessionID)

	// Step 3: Pack as little-endian uint32
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, k)

	// Step 4: XOR with "ZKSO"
	xorKeys := []byte{'Z', 'K', 'S', 'O'}
	for i := 0; i < 4; i++ {
		b[i] ^= xorKeys[i]
	}

	// Step 5: Swap uint16 halves (unpack as 2 shorts, swap, repack)
	s1 := binary.LittleEndian.Uint16(b[0:2])
	s2 := binary.LittleEndian.Uint16(b[2:4])
	binary.LittleEndian.PutUint16(b[0:2], s2)
	binary.LittleEndian.PutUint16(b[2:4], s1)

	// Step 6: XOR with mask (0xFF & 50 = 50 = 0x32)
	mask := byte(0xFF & 50)
	b[0] ^= mask
	b[1] ^= mask
	b[2] = mask
	b[3] ^= mask

	return b
}

// encodeTime encodes a time.Time to ZKTeco packed timestamp
func encodeTime(t time.Time) uint32 {
	y := t.Year() % 100
	m := int(t.Month())
	d := t.Day()
	h := t.Hour()
	min := t.Minute()
	sec := t.Second()
	return uint32(((y*12*31+(m-1)*31+d-1)*24*60*60 + (h*60+min)*60 + sec))
}

// decodeTime decodes a ZKTeco packed timestamp to time.Time
func decodeTime(t uint32) time.Time {
	second := int(t % 60)
	t /= 60
	minute := int(t % 60)
	t /= 60
	hour := int(t % 24)
	t /= 24
	day := int(t%31 + 1)
	t /= 31
	month := int(t%12 + 1)
	t /= 12
	year := int(t + 2000)
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
}

// reverseHex reverses hex string in 2-character chunks (byte-reversal)
func reverseHex(hexStr string) string {
	result := ""
	for i := len(hexStr) - 2; i >= 0; i -= 2 {
		result += hexStr[i : i+2]
	}
	return result
}
