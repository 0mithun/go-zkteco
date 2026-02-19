package zkteco

// Command codes
const (
	CMD_CONNECT        = 1000
	CMD_EXIT           = 1001
	CMD_ENABLE_DEVICE  = 1002
	CMD_DISABLE_DEVICE = 1003
	CMD_RESTART        = 1004
	CMD_POWEROFF       = 1005
	CMD_SLEEP          = 1006
	CMD_RESUME         = 1007
	CMD_TEST_TEMP      = 1011
	CMD_TESTVOICE      = 1017
	CMD_CHANGE_SPEED   = 1101

	CMD_WRITE_LCD = 66
	CMD_CLEAR_LCD = 67

	CMD_ACK_OK     = 2000
	CMD_ACK_ERROR  = 2001
	CMD_ACK_DATA   = 2002
	CMD_ACK_UNAUTH = 2005
	CMD_ACK_AUTH   = 1102

	CMD_PREPARE_DATA = 1500
	CMD_DATA         = 1501
	CMD_FREE_DATA    = 1502

	CMD_USER_TEMP_RRQ    = 9
	CMD_USER_TEMP_WRQ    = 10
	CMD_DEVICE           = 11
	CMD_OPTIONS_WRQ      = 12
	CMD_ATT_LOG_RRQ      = 13
	CMD_CLEAR_DATA       = 14
	CMD_CLEAR_ATT_LOG    = 15
	CMD_DELETE_USER      = 18
	CMD_DELETE_USER_TEMP = 19
	CMD_CLEAR_ADMIN      = 20
	CMD_GET_FREE_SIZES   = 50

	CMD_GET_TIME = 201
	CMD_SET_TIME = 202

	CMD_REG_EVENT = 500
	CMD_VERSION   = 1100
	CMD_SET_USER  = 8
)

// Function types for CMD_USER_TEMP_RRQ
const (
	FCT_ATTLOG    = 1
	FCT_FINGERTMP = 2
	FCT_OPLOG     = 4
	FCT_USER      = 5
	FCT_SMS       = 6
	FCT_UDATA     = 7
	FCT_WORKCODE  = 8
)

// User roles
const (
	LEVEL_USER  = 0
	LEVEL_ADMIN = 14
)

// Attendance states
const (
	STATE_PASSWORD    = 0
	STATE_FINGERPRINT = 1
	STATE_CARD        = 2
)

// Attendance types
const (
	TYPE_CHECK_IN     = 0
	TYPE_CHECK_OUT    = 1
	TYPE_BREAK_IN     = 2
	TYPE_BREAK_OUT    = 3
	TYPE_OVERTIME_IN  = 4
	TYPE_OVERTIME_OUT = 5
)

// Event flags for CMD_REG_EVENT
const (
	EF_ATTLOG       = 1
	EF_FINGER       = 2
	EF_ENROLLUSER   = 4
	EF_ENROLLFINGER = 8
	EF_BUTTON       = 16
	EF_UNLOCK       = 32
	EF_VERIFY       = 128
	EF_FPFTR        = 256
	EF_ALARM        = 512
)

// StateName returns a human-readable name for an attendance state.
func StateName(state int) string {
	switch state {
	case STATE_PASSWORD:
		return "Password"
	case STATE_FINGERPRINT:
		return "Fingerprint"
	case STATE_CARD:
		return "Card"
	default:
		return "Unknown"
	}
}

// TypeName returns a human-readable name for an attendance type.
func TypeName(typ int) string {
	switch typ {
	case TYPE_CHECK_IN:
		return "Check-In"
	case TYPE_CHECK_OUT:
		return "Check-Out"
	case TYPE_BREAK_IN:
		return "Break-In"
	case TYPE_BREAK_OUT:
		return "Break-Out"
	case TYPE_OVERTIME_IN:
		return "OT-In"
	case TYPE_OVERTIME_OUT:
		return "OT-Out"
	default:
		return "Unknown"
	}
}
