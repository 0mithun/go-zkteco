package zkteco

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// RealTimeEvent represents a real-time event from the device.
type RealTimeEvent struct {
	EventType   int       `json:"event_type"`
	EventName   string    `json:"event_name"`
	UserID      string    `json:"user_id,omitempty"`
	Time        time.Time `json:"time,omitempty"`
	State       int       `json:"state,omitempty"`
	DeviceIP    string    `json:"device_ip,omitempty"`
	RawData     []byte    `json:"raw_data,omitempty"`
	FingerIndex int       `json:"finger_index,omitempty"`
	ButtonID    int       `json:"button_id,omitempty"`
	DoorID      int       `json:"door_id,omitempty"`
	UnlockType  int       `json:"unlock_type,omitempty"`
	AlarmType   int       `json:"alarm_type,omitempty"`
}

// EventCallback is called when a real-time event is received.
type EventCallback func(event RealTimeEvent)

// GetRealTimeLogs listens for real-time attendance log events.
func (z *ZKTeco) GetRealTimeLogs(callback EventCallback, timeout time.Duration) error {
	return z.GetRealTimeEvents(callback, EF_ATTLOG, timeout)
}

// GetRealTimeEvents listens for real-time events matching the event mask.
func (z *ZKTeco) GetRealTimeEvents(callback EventCallback, eventMask int, timeout time.Duration) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(eventMask))

	resp, err := z.command(CMD_REG_EVENT, data, "general")
	if err != nil {
		return fmt.Errorf("register events: %w", err)
	}

	pkt, err := parsePacket(resp)
	if err != nil {
		return fmt.Errorf("parse reg event response: %w", err)
	}
	if pkt.Command != CMD_ACK_OK {
		return fmt.Errorf("register events: error response %d", pkt.Command)
	}

	startTime := time.Now()

	for {
		if timeout > 0 && time.Since(startTime) >= timeout {
			break
		}

		readTimeout := 1 * time.Second
		if timeout > 0 {
			remaining := timeout - time.Since(startTime)
			if remaining < readTimeout {
				readTimeout = remaining
			}
		}
		z.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var payload []byte
		if z.IsTCP() {
			payload, err = z.recvTCP()
		} else {
			payload, err = z.recvUDP()
		}

		if err != nil {
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				continue
			}
			return fmt.Errorf("receive event: %w", err)
		}

		if len(payload) < 4 {
			continue
		}

		cmdID := binary.LittleEndian.Uint16(payload[0:2])
		if cmdID != CMD_REG_EVENT {
			continue
		}

		if len(payload) < 6 {
			continue
		}

		eventType := int(binary.LittleEndian.Uint16(payload[4:6]))

		if eventType&eventMask == 0 {
			continue
		}

		event := z.decodeRealTimeEvent(payload, eventType)
		callback(event)
	}

	return nil
}

func (z *ZKTeco) decodeRealTimeEvent(payload []byte, eventType int) RealTimeEvent {
	event := RealTimeEvent{
		EventType: eventType,
		EventName: EventName(eventType),
		DeviceIP:  z.host,
		Time:      time.Now(),
	}

	if len(payload) <= 8 {
		event.RawData = payload
		return event
	}

	recvData := payload[8:]

	switch eventType {
	case EF_ATTLOG:
		event = z.decodeAttLogEvent(recvData, event)
	case EF_ENROLLUSER, EF_VERIFY:
		if len(recvData) >= 9 {
			event.UserID = strings.TrimRight(string(recvData[0:9]), "\x00")
		}
	case EF_FINGER, EF_ENROLLFINGER, EF_FPFTR:
		if len(recvData) >= 10 {
			event.UserID = strings.TrimRight(string(recvData[0:9]), "\x00")
			event.FingerIndex = int(recvData[9])
		}
	case EF_BUTTON:
		if len(recvData) >= 2 {
			event.ButtonID = int(binary.LittleEndian.Uint16(recvData[0:2]))
		}
	case EF_UNLOCK:
		if len(recvData) >= 2 {
			event.DoorID = int(recvData[0])
			event.UnlockType = int(recvData[1])
		}
	case EF_ALARM:
		if len(recvData) >= 2 {
			event.AlarmType = int(binary.LittleEndian.Uint16(recvData[0:2]))
		}
	default:
		event.RawData = recvData
	}

	return event
}

func (z *ZKTeco) decodeAttLogEvent(recvData []byte, event RealTimeEvent) RealTimeEvent {
	if len(recvData) < 32 {
		event.RawData = recvData
		return event
	}

	event.UserID = strings.TrimRight(string(recvData[0:9]), "\x00")

	if len(recvData) > 24 {
		event.State = int(recvData[24])
	}

	if len(recvData) >= 32 {
		year := 2000 + int(recvData[26])
		month := int(recvData[27])
		day := int(recvData[28])
		hour := int(recvData[29])
		minute := int(recvData[30])
		second := int(recvData[31])

		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			event.Time = time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
		}
	}

	return event
}

// EventName returns a human-readable name for an event type.
func EventName(eventType int) string {
	switch eventType {
	case EF_ATTLOG:
		return "attendance"
	case EF_FINGER:
		return "finger"
	case EF_ENROLLUSER:
		return "enroll_user"
	case EF_ENROLLFINGER:
		return "enroll_finger"
	case EF_BUTTON:
		return "button"
	case EF_UNLOCK:
		return "unlock"
	case EF_VERIFY:
		return "verify"
	case EF_FPFTR:
		return "finger_feature"
	case EF_ALARM:
		return "alarm"
	default:
		return "unknown"
	}
}
