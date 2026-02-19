package zkteco

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// getDeviceOption sends CMD_DEVICE with a key and returns the value.
func (z *ZKTeco) getDeviceOption(key string) (string, error) {
	resp, err := z.command(CMD_DEVICE, []byte(key), "general")
	if err != nil {
		return "", err
	}

	pkt, err := parsePacket(resp)
	if err != nil {
		return "", err
	}

	if pkt.Command != CMD_ACK_OK && pkt.Command != CMD_ACK_DATA {
		return "", fmt.Errorf("device option %q: error response %d", key, pkt.Command)
	}

	value := string(pkt.Data)
	if idx := strings.Index(value, "="); idx >= 0 {
		value = value[idx+1:]
	}
	value = strings.TrimRight(value, "\x00")
	return value, nil
}

// Version returns the firmware version.
func (z *ZKTeco) Version() (string, error) {
	resp, err := z.command(CMD_VERSION, nil, "general")
	if err != nil {
		return "", err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(pkt.Data), "\x00"), nil
}

// SerialNumber returns the device serial number.
func (z *ZKTeco) SerialNumber() (string, error) {
	return z.getDeviceOption("~SerialNumber")
}

// DeviceName returns the device name.
func (z *ZKTeco) DeviceName() (string, error) {
	return z.getDeviceOption("~DeviceName")
}

// DeviceID returns the device ID.
func (z *ZKTeco) DeviceID() (string, error) {
	return z.getDeviceOption("DeviceID")
}

// VendorName returns the vendor/OEM name.
func (z *ZKTeco) VendorName() (string, error) {
	return z.getDeviceOption("~OEMVendor")
}

// Platform returns the device platform.
func (z *ZKTeco) Platform() (string, error) {
	return z.getDeviceOption("~Platform")
}

// OSVersion returns the OS version.
func (z *ZKTeco) OSVersion() (string, error) {
	return z.getDeviceOption("~OS")
}

// FMVersion returns the fingerprint module version.
func (z *ZKTeco) FMVersion() (string, error) {
	return z.getDeviceOption("~ZKFPVersion")
}

// SSR returns the SSR info.
func (z *ZKTeco) SSR() (string, error) {
	return z.getDeviceOption("~SSR")
}

// PinWidth returns the PIN width.
func (z *ZKTeco) PinWidth() (string, error) {
	return z.getDeviceOption("~PIN2Width")
}

// FaceFunctionOn returns the face function status.
func (z *ZKTeco) FaceFunctionOn() (string, error) {
	return z.getDeviceOption("FaceFunOn")
}

// WorkCode returns the work code info.
func (z *ZKTeco) WorkCode() (string, error) {
	return z.getDeviceOption("WorkCode")
}

// MemoryInfo holds device memory/capacity information.
type MemoryInfo struct {
	AdminCount   int
	UserCount    int
	UserCapacity int
	LogCount     int
	LogCapacity  int
}

// GetMemoryInfo returns memory usage and capacity info.
func (z *ZKTeco) GetMemoryInfo() (*MemoryInfo, error) {
	resp, err := z.command(CMD_GET_FREE_SIZES, nil, "general")
	if err != nil {
		return nil, err
	}

	pkt, err := parsePacket(resp)
	if err != nil {
		return nil, err
	}

	if pkt.Command != CMD_ACK_OK && pkt.Command != CMD_ACK_DATA {
		return nil, fmt.Errorf("getMemoryInfo: error response %d", pkt.Command)
	}

	data := pkt.Data
	if len(data) < 68 {
		return nil, fmt.Errorf("getMemoryInfo: response too short: %d bytes", len(data))
	}

	info := &MemoryInfo{}
	if len(data) > 51 {
		info.AdminCount = int(binary.LittleEndian.Uint32(data[48:52]))
	}
	if len(data) > 19 {
		info.UserCount = int(binary.LittleEndian.Uint32(data[16:20]))
	}
	if len(data) > 63 {
		info.UserCapacity = int(binary.LittleEndian.Uint32(data[60:64]))
	}
	if len(data) > 35 {
		info.LogCount = int(binary.LittleEndian.Uint32(data[32:36]))
	}
	if len(data) > 67 {
		info.LogCapacity = int(binary.LittleEndian.Uint32(data[64:68]))
	}

	return info, nil
}

// GetDeviceData gets a raw device option by key.
func (z *ZKTeco) GetDeviceData(key string) (string, error) {
	return z.getDeviceOption(key)
}

// SetCustomData sets a custom key-value pair on the device.
func (z *ZKTeco) SetCustomData(key, value string) error {
	data := []byte(fmt.Sprintf("*%s=%s", key, value))
	resp, err := z.command(CMD_OPTIONS_WRQ, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("setCustomData: error response %d", pkt.Command)
	}
	return nil
}

// GetCustomData gets a custom key-value pair from the device.
func (z *ZKTeco) GetCustomData(key string) (string, error) {
	return z.getDeviceOption("*" + key)
}

// SetPushCommKey sets the push communication key.
func (z *ZKTeco) SetPushCommKey(value string) error {
	data := []byte(fmt.Sprintf("pushcommkey=%s", value))
	resp, err := z.command(CMD_OPTIONS_WRQ, data, "general")
	if err != nil {
		return err
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("setPushCommKey: error response %d", pkt.Command)
	}
	return nil
}

// GetPushCommKey gets the push communication key.
func (z *ZKTeco) GetPushCommKey() (string, error) {
	return z.getDeviceOption("pushcommkey")
}
