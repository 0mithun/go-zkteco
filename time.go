package zkteco

import (
	"encoding/binary"
	"fmt"
	"time"
)

// GetTime returns the device time.
func (z *ZKTeco) GetTime() (time.Time, error) {
	resp, err := z.command(CMD_GET_TIME, nil, "general")
	if err != nil {
		return time.Time{}, err
	}

	pkt, err := parsePacket(resp)
	if err != nil {
		return time.Time{}, err
	}

	if len(pkt.Data) < 4 {
		return time.Time{}, fmt.Errorf("getTime: response too short")
	}

	encoded := binary.LittleEndian.Uint32(pkt.Data[0:4])
	return decodeTime(encoded), nil
}

// SetTime sets the device time.
func (z *ZKTeco) SetTime(t time.Time) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, encodeTime(t))

	resp, err := z.command(CMD_SET_TIME, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("setTime: error response %d", pkt.Command)
	}
	return nil
}
