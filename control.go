package zkteco

import (
	"encoding/binary"
	"fmt"
)

// EnableDevice enables the device (resumes normal operation).
func (z *ZKTeco) EnableDevice() error {
	resp, err := z.command(CMD_ENABLE_DEVICE, nil, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("enableDevice: error response %d", pkt.Command)
	}
	return nil
}

// DisableDevice disables the device (shows "working..." on screen).
func (z *ZKTeco) DisableDevice() error {
	data := []byte{0x00, 0x00}
	resp, err := z.command(CMD_DISABLE_DEVICE, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("disableDevice: error response %d", pkt.Command)
	}
	return nil
}

// Restart restarts the device.
func (z *ZKTeco) Restart() error {
	data := []byte{0x00, 0x00}
	resp, err := z.command(CMD_RESTART, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("restart: error response %d", pkt.Command)
	}
	return nil
}

// Shutdown powers off the device.
func (z *ZKTeco) Shutdown() error {
	data := []byte{0x00, 0x00}
	resp, err := z.command(CMD_POWEROFF, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("shutdown: error response %d", pkt.Command)
	}
	return nil
}

// Sleep puts the device to sleep.
func (z *ZKTeco) Sleep() error {
	data := []byte{0x00, 0x00}
	resp, err := z.command(CMD_SLEEP, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("sleep: error response %d", pkt.Command)
	}
	return nil
}

// Resume wakes the device from sleep.
func (z *ZKTeco) Resume() error {
	data := []byte{0x00, 0x00}
	resp, err := z.command(CMD_RESUME, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("resume: error response %d", pkt.Command)
	}
	return nil
}

// TestVoice plays a voice/sound by index.
func (z *ZKTeco) TestVoice(index int) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(index))
	resp, err := z.command(CMD_TESTVOICE, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("testVoice: error response %d", pkt.Command)
	}
	return nil
}

// WriteLCD writes a message to the device LCD display.
func (z *ZKTeco) WriteLCD(message string) error {
	rank := 2
	data := make([]byte, 0, 4+len(message))
	data = append(data, byte(rank), byte(rank>>8), 0x00, ' ')
	data = append(data, []byte(message)...)

	resp, err := z.command(CMD_WRITE_LCD, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("writeLCD: error response %d", pkt.Command)
	}
	return nil
}

// ClearLCD clears the LCD display.
func (z *ZKTeco) ClearLCD() error {
	resp, err := z.command(CMD_CLEAR_LCD, nil, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("clearLCD: error response %d", pkt.Command)
	}
	return nil
}
