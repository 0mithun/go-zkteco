package zkteco

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Attendance represents an attendance record from the device.
type Attendance struct {
	UID        int       `json:"uid"`
	UserID     string    `json:"user_id"`
	State      int       `json:"state"`
	RecordTime time.Time `json:"record_time"`
	Type       int       `json:"type"`
}

// GetAttendances retrieves all attendance records from the device.
func (z *ZKTeco) GetAttendances() ([]Attendance, error) {
	allData, err := z.commandData(CMD_ATT_LOG_RRQ, nil)
	if err != nil {
		return nil, fmt.Errorf("getAttendances: %w", err)
	}

	if len(allData) <= 8 {
		return nil, nil
	}

	// Skip first 10 bytes (8 header + 2 extra) — matches PHP behavior
	data := allData
	if len(data) > 10 {
		data = data[10:]
	}

	// Each attendance record is 40 bytes
	recordSize := 40
	var records []Attendance

	for i := 0; i+recordSize <= len(data); i += recordSize {
		rec := data[i : i+recordSize]
		att := parseAttendanceRecord(rec)
		if att != nil {
			records = append(records, *att)
		}
	}

	return records, nil
}

// parseAttendanceRecord parses a 40-byte attendance record.
// Uses the same hex-based parsing as the PHP package for compatibility.
func parseAttendanceRecord(rec []byte) *Attendance {
	if len(rec) < 39 {
		return nil
	}

	hexStr := hex.EncodeToString(rec[:39])
	if len(hexStr) < 68 {
		return nil
	}

	// UID: bytes 2-3 (hex offset 4-7)
	uidLo, _ := strconv.ParseInt(hexStr[4:6], 16, 64)
	uidHi, _ := strconv.ParseInt(hexStr[6:8], 16, 64)
	uid := int(uidHi*256 + uidLo)
	if uid == 0 {
		return nil
	}

	// UserID: bytes 4-12 (hex offset 8-25), 9 bytes ASCII
	userIDBytes := rec[4:13]
	userID := strings.TrimRight(string(userIDBytes), "\x00")

	// State: byte 28 (hex offset 56-57)
	state, _ := strconv.ParseInt(hexStr[56:58], 16, 64)

	// Timestamp: bytes 29-32 (hex offset 58-65), 4 bytes LE → decode time
	timeHex := hexStr[58:66]
	reversed := reverseHex(timeHex)
	timeVal, _ := strconv.ParseUint(reversed, 16, 32)
	recordTime := decodeTime(uint32(timeVal))

	// Type: byte 33 (hex offset 66-67)
	typ, _ := strconv.ParseInt(hexStr[66:68], 16, 64)

	return &Attendance{
		UID:        uid,
		UserID:     userID,
		State:      int(state),
		RecordTime: recordTime,
		Type:       int(typ),
	}
}

// ClearAttendance clears all attendance records.
// WARNING: This is destructive!
func (z *ZKTeco) ClearAttendance() error {
	resp, err := z.command(CMD_CLEAR_ATT_LOG, nil, "general")
	if err != nil {
		return fmt.Errorf("clearAttendance: %w", err)
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("clearAttendance: error response %d", pkt.Command)
	}
	return nil
}

// GetFingerprints retrieves fingerprint data for a user.
func (z *ZKTeco) GetFingerprints(uid int) (map[int][]byte, error) {
	result := make(map[int][]byte)

	for finger := 0; finger <= 9; finger++ {
		data := []byte{byte(uid & 0xFF), byte((uid >> 8) & 0xFF), byte(finger)}
		allData, err := z.commandData(CMD_USER_TEMP_RRQ, data)
		if err != nil {
			continue // No fingerprint for this finger
		}

		if len(allData) <= 8 {
			continue
		}

		// Extract fingerprint template data
		pkt, err := parsePacket(allData)
		if err != nil || pkt == nil {
			continue
		}

		if len(pkt.Data) > 6 {
			// Fingerprint template has size(2) + uid(2) + finger(1) + flag(1) + templateData
			templateSize := int(binary.LittleEndian.Uint16(pkt.Data[0:2]))
			if templateSize > 0 && len(pkt.Data) >= 6+templateSize {
				template := make([]byte, templateSize)
				copy(template, pkt.Data[6:6+templateSize])
				result[finger] = template
			}
		}
	}

	return result, nil
}
