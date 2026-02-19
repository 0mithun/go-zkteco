package zkteco

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// User represents a user record from the device.
type User struct {
	UID      int    `json:"uid"`
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     int    `json:"role"`
	CardNo   int    `json:"card_no"`
}

// GetUsers retrieves all users from the device.
func (z *ZKTeco) GetUsers() ([]User, error) {
	cmdData := []byte{FCT_USER}
	allData, err := z.commandData(CMD_USER_TEMP_RRQ, cmdData)
	if err != nil {
		return nil, fmt.Errorf("getUsers: %w", err)
	}

	if len(allData) <= 8 {
		return nil, nil
	}

	data := allData[8:]

	recordSize := 72
	var users []User

	for i := 0; i+recordSize <= len(data); i += recordSize {
		rec := data[i : i+recordSize]
		user := parseUserRecord(rec)
		if user != nil {
			users = append(users, *user)
		}
	}

	return users, nil
}

// parseUserRecord parses a 72-byte user record.
func parseUserRecord(rec []byte) *User {
	if len(rec) < 72 {
		return nil
	}

	uid := int(binary.LittleEndian.Uint16(rec[1:3]))
	role := int(rec[3])
	password := strings.TrimRight(string(rec[4:12]), "\x00")
	name := strings.TrimRight(string(rec[12:36]), "\x00")
	cardNo := int(binary.LittleEndian.Uint32(rec[36:40]))
	userID := strings.TrimRight(string(rec[49:72]), "\x00")

	return &User{
		UID:      uid,
		UserID:   userID,
		Name:     name,
		Password: password,
		Role:     role,
		CardNo:   cardNo,
	}
}

// SetUser creates or updates a user on the device.
func (z *ZKTeco) SetUser(uid int, userID string, name string, password string, role int, cardNo int) error {
	data := make([]byte, 72)

	data[0] = byte(uid & 0xFF)
	data[1] = byte((uid >> 8) & 0xFF)
	data[2] = byte(role)

	copy(data[3:11], make([]byte, 8))
	if len(password) > 8 {
		password = password[:8]
	}
	copy(data[3:], []byte(password))

	copy(data[11:35], make([]byte, 24))
	if len(name) > 24 {
		name = name[:24]
	}
	copy(data[11:], []byte(name))

	binary.LittleEndian.PutUint32(data[35:39], uint32(cardNo))

	data[39] = 1

	if len(userID) > 9 {
		userID = userID[:9]
	}
	copy(data[48:57], make([]byte, 9))
	copy(data[48:], []byte(userID))

	resp, err := z.command(CMD_SET_USER, data, "general")
	if err != nil {
		return fmt.Errorf("setUser: %w", err)
	}

	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("setUser: error response %d", pkt.Command)
	}
	return nil
}

// RemoveUser removes a user by UID.
func (z *ZKTeco) RemoveUser(uid int) error {
	data := []byte{byte(uid & 0xFF), byte((uid >> 8) & 0xFF)}
	resp, err := z.command(CMD_DELETE_USER, data, "general")
	if err != nil {
		return fmt.Errorf("removeUser: %w", err)
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("removeUser: error response %d", pkt.Command)
	}
	return nil
}

// ClearAllUsers clears ALL data on the device.
func (z *ZKTeco) ClearAllUsers() error {
	resp, err := z.command(CMD_CLEAR_DATA, nil, "general")
	if err != nil {
		return fmt.Errorf("clearAllUsers: %w", err)
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("clearAllUsers: error response %d", pkt.Command)
	}
	return nil
}

// ClearAdmin removes admin privileges from all users.
func (z *ZKTeco) ClearAdmin() error {
	resp, err := z.command(CMD_CLEAR_ADMIN, nil, "general")
	if err != nil {
		return fmt.Errorf("clearAdmin: %w", err)
	}
	pkt, err := parsePacket(resp)
	if err != nil {
		return err
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("clearAdmin: error response %d", pkt.Command)
	}
	return nil
}
