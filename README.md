
</p>
<p align="center">
    <img src="https://raw.githubusercontent.com/0mithun/go-zkteco/main/docs/device.png" alt="ZKTeco Device" width="400"/>
</p>

# go-zkteco

[![Go Reference](https://pkg.go.dev/badge/github.com/0mithun/go-zkteco.svg)](https://pkg.go.dev/github.com/0mithun/go-zkteco)
[![Go Report Card](https://goreportcard.com/badge/github.com/0mithun/go-zkteco)](https://goreportcard.com/report/github.com/0mithun/go-zkteco)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A pure Go client library for **ZKTeco** biometric attendance devices. Implements the ZKTeco proprietary binary protocol over TCP and UDP — zero external dependencies, fully compatible with the [0mithun/php-zkteco](https://github.com/0mithun/php-zkteco) PHP package.

## Features

- **Dual Protocol** — TCP and UDP support
- **Password Authentication** — Connect to password-protected devices
- **TCPMUX Proxy** — HTTP CONNECT proxy support for subdomain-based device routing
- **User Management** — Get, set, remove users and admin privileges
- **Attendance Logs** — Retrieve and clear attendance records
- **Real-Time Events** — Live attendance punch monitoring with event callbacks
- **Fingerprint Templates** — Read fingerprint data per user
- **Device Control** — Enable, disable, restart, shutdown, sleep, resume
- **LCD & Voice** — Write text to LCD, play voice prompts
- **Device Info** — Serial number, firmware, platform, memory info
- **Custom Data** — Read/write arbitrary key-value pairs on device
- **Zero Dependencies** — Only Go standard library

## Requirements

- Go 1.22+

## Installation

```bash
go get github.com/0mithun/go-zkteco
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    zkteco "github.com/0mithun/go-zkteco"
)

func main() {
    zk := zkteco.NewZKTeco("192.168.1.201", 4370,
        zkteco.WithProtocol("tcp"),
        zkteco.WithTimeout(25),
    )

    if err := zk.Connect(); err != nil {
        log.Fatal(err)
    }
    defer zk.Disconnect()

    // Get users
    users, _ := zk.GetUsers()
    fmt.Printf("Found %d users\n", len(users))

    // Get attendance logs
    records, _ := zk.GetAttendances()
    fmt.Printf("Found %d attendance records\n", len(records))
}
```

## Constructor & Options

```go
zk := zkteco.NewZKTeco(host, port, opts...)
```

| Option | Default | Description |
|--------|---------|-------------|
| `WithProtocol("tcp")` | `"udp"` | Connection protocol: `"tcp"` or `"udp"` |
| `WithTimeout(30)` | `25` | Socket timeout in seconds |
| `WithPassword(123456)` | `0` | Device communication password |
| `WithTCPMUX(host, port, subdomain)` | disabled | TCPMUX HTTP CONNECT proxy (forces TCP) |

## TCPMUX HTTP CONNECT Proxy

For devices behind a reverse proxy (e.g., FRP with `tcpmux_httpconnect`), use `WithTCPMUX`:

```go
zk := zkteco.NewZKTeco("192.168.1.201", 4370,
    zkteco.WithTCPMUX("proxy.example.com", 1337, "device1"),
    zkteco.WithTimeout(30),
)
```

The client performs an HTTP CONNECT handshake through the proxy before initiating the ZKTeco protocol.

| | Direct Connection | TCPMUX Proxy |
|---|---|---|
| **Connection** | Device IP:Port | Proxy IP:Port |
| **Protocol** | TCP or UDP | TCP only |
| **Use Case** | LAN / direct access | NAT / cloud proxy |

## API Reference

### Connection

```go
// Connect to the device
err := zk.Connect()

// Disconnect
err := zk.Disconnect()

// Check protocol
isTCP := zk.IsTCP()
```

### Device Information

```go
serial, err := zk.SerialNumber()    // "PAS4234400018"
name, err := zk.DeviceName()        // "K40"
id, err := zk.DeviceID()            // "1"
vendor, err := zk.VendorName()      // "ZKTECO CO., LTD."
platform, err := zk.Platform()      // "ZLM60_TFT"
version, err := zk.Version()        // "Ver 6.60 Apr 13 2022"
osVer, err := zk.OSVersion()        // "1"
fmVer, err := zk.FMVersion()        // firmware module version
ssr, err := zk.SSR()                // SSR capability
pinWidth, err := zk.PinWidth()      // PIN width setting
faceOn, err := zk.FaceFunctionOn()  // face recognition status
workCode, err := zk.WorkCode()      // work code setting
```

### Memory Info

```go
info, err := zk.GetMemoryInfo()
fmt.Printf("Users: %d/%d\n", info.UserCount, info.UserCapacity)
fmt.Printf("Logs:  %d/%d\n", info.LogCount, info.LogCapacity)
fmt.Printf("Admins: %d\n", info.AdminCount)
```

**`MemoryInfo` struct:**

| Field | Type | Description |
|-------|------|-------------|
| `AdminCount` | `int` | Number of admin users |
| `UserCount` | `int` | Number of enrolled users |
| `UserCapacity` | `int` | Maximum user capacity |
| `LogCount` | `int` | Number of attendance logs |
| `LogCapacity` | `int` | Maximum log capacity |

### Time Management

```go
// Get device time
t, err := zk.GetTime()
fmt.Println(t.Format("2006-01-02 15:04:05"))

// Set device time
err := zk.SetTime(time.Now())
```

### User Management

```go
// Get all users
users, err := zk.GetUsers()
for _, u := range users {
    fmt.Printf("UID=%d ID=%s Name=%s Role=%d\n",
        u.UID, u.UserID, u.Name, u.Role)
}

// Create or update a user
err := zk.SetUser(
    1,           // uid
    "101",       // userID
    "John Doe",  // name
    "1234",      // password
    zkteco.LEVEL_USER,  // role (LEVEL_USER=0, LEVEL_ADMIN=14)
    0,           // cardNo
)

// Remove a user
err := zk.RemoveUser(1) // by UID

// Clear ALL data (users, attendance, fingerprints)
err := zk.ClearAllUsers()

// Remove admin privileges (demote all admins to users)
err := zk.ClearAdmin()
```

**`User` struct:**

| Field | Type | JSON | Description |
|-------|------|------|-------------|
| `UID` | `int` | `uid` | Internal device UID |
| `UserID` | `string` | `user_id` | User ID string |
| `Name` | `string` | `name` | User display name |
| `Password` | `string` | `password` | User password |
| `Role` | `int` | `role` | 0=User, 14=Admin |
| `CardNo` | `int` | `card_no` | RFID card number |

### Attendance Logs

```go
// Get all attendance records
records, err := zk.GetAttendances()
for _, a := range records {
    fmt.Printf("User=%s Time=%s State=%s Type=%s\n",
        a.UserID,
        a.RecordTime.Format("2006-01-02 15:04:05"),
        zkteco.StateName(a.State),
        zkteco.TypeName(a.Type),
    )
}

// Clear all attendance logs
err := zk.ClearAttendance()
```

**`Attendance` struct:**

| Field | Type | JSON | Description |
|-------|------|------|-------------|
| `UID` | `int` | `uid` | Internal device UID |
| `UserID` | `string` | `user_id` | User ID string |
| `State` | `int` | `state` | 0=Password, 1=Fingerprint, 2=Card |
| `RecordTime` | `time.Time` | `record_time` | Timestamp of the punch |
| `Type` | `int` | `type` | 0=CheckIn, 1=CheckOut, 2=BreakIn, etc. |

### Real-Time Events

Listen for live attendance punches as they happen:

```go
callback := func(event zkteco.RealTimeEvent) {
    fmt.Printf("User %s punched at %s (state=%d)\n",
        event.UserID,
        event.Time.Format("15:04:05"),
        event.State,
    )
}

// Listen for attendance events for 60 seconds
err := zk.GetRealTimeLogs(callback, 60*time.Second)

// Listen for specific event types (bitmask)
err := zk.GetRealTimeEvents(callback,
    zkteco.EF_ATTLOG|zkteco.EF_FINGER|zkteco.EF_VERIFY,
    60*time.Second,
)

// Listen indefinitely (timeout=0)
err := zk.GetRealTimeLogs(callback, 0)
```

**`RealTimeEvent` struct:**

| Field | Type | Description |
|-------|------|-------------|
| `EventType` | `int` | Event flag that triggered this event |
| `EventName` | `string` | Human-readable: `"attendance"`, `"finger"`, etc. |
| `UserID` | `string` | User who triggered the event |
| `Time` | `time.Time` | Event timestamp from device |
| `State` | `int` | Attendance state (check-in/out) |
| `DeviceIP` | `string` | IP of the device |
| `FingerIndex` | `int` | Finger index (for finger events) |
| `ButtonID` | `int` | Button ID (for button events) |
| `DoorID` | `int` | Door ID (for unlock events) |
| `UnlockType` | `int` | Unlock type (for unlock events) |
| `AlarmType` | `int` | Alarm type (for alarm events) |
| `RawData` | `[]byte` | Raw event data for custom parsing |

**Event Flags:**

| Constant | Value | Description |
|----------|-------|-------------|
| `EF_ATTLOG` | 1 | Attendance log |
| `EF_FINGER` | 2 | Fingerprint placed |
| `EF_ENROLLUSER` | 4 | User enrolled |
| `EF_ENROLLFINGER` | 8 | Fingerprint enrolled |
| `EF_BUTTON` | 16 | Button pressed |
| `EF_UNLOCK` | 32 | Door unlocked |
| `EF_VERIFY` | 128 | Verification event |
| `EF_FPFTR` | 256 | Fingerprint feature |
| `EF_ALARM` | 512 | Alarm triggered |

### Fingerprint Templates

```go
// Get fingerprint templates for a user (by UID)
fingerprints, err := zk.GetFingerprints(1)
for fingerIdx, templateData := range fingerprints {
    fmt.Printf("Finger %d: %d bytes\n", fingerIdx, len(templateData))
}
```

### Device Control

```go
err := zk.EnableDevice()   // Enable the device
err := zk.DisableDevice()  // Disable (lock) the device
err := zk.Restart()        // Restart the device
err := zk.Shutdown()       // Power off the device
err := zk.Sleep()          // Enter sleep mode
err := zk.Resume()         // Wake from sleep
```

### LCD Display & Voice

```go
// Write text to the LCD screen
err := zk.WriteLCD("Hello World!")

// Clear the LCD screen
err := zk.ClearLCD()

// Play a voice prompt (0-55)
err := zk.TestVoice(0)  // "Thank You"
err := zk.TestVoice(1)  // "Incorrect Password"
```

**Voice Index Reference:**

| Index | Voice |
|-------|-------|
| 0 | Thank You |
| 1 | Incorrect Password |
| 2 | Access Denied |
| 3 | Invalid ID |
| 4 | Please try again |
| 5 | Duplicate ID |
| 6 | The clock is flow |
| 7 | The clock is full |
| 8 | Duplicate finger |
| 9 | Duplicated punch |
| 10 | Beep kuko |
| 11 | Beep siren |
| 13 | Beep bell |
| 24 | Beep standard |
| 30 | Invalid user |
| 36 | Fingerprint not registered |
| 51 | Focus eyes on the green box |

### Custom Data

```go
// Set a custom key-value pair
err := zk.SetCustomData("myKey", "myValue")

// Get a custom key-value pair
val, err := zk.GetCustomData("myKey")

// Push comm key
err := zk.SetPushCommKey("secretKey123")
key, err := zk.GetPushCommKey()

// Get any device option by key
data, err := zk.GetDeviceData("~DeviceName")
```

## Password Authentication

When a device has a communication password set, connect with `WithPassword`:

```go
zk := zkteco.NewZKTeco("192.168.1.201", 4370,
    zkteco.WithProtocol("tcp"),
    zkteco.WithPassword(123456),
)

err := zk.Connect() // Automatically handles CMD_ACK_UNAUTH → CMD_ACK_AUTH
```

If the password is wrong, `Connect()` returns an error:  
`authentication failed: command=2005`

## Protocol Comparison

| Feature | TCP | UDP |
|---------|-----|-----|
| **Max Packet** | 65535 bytes | 65535 bytes |
| **Header** | 8-byte TCP prefix + 8-byte header | 8-byte header |
| **Reliability** | Built-in | Application-level |
| **Use Case** | Recommended for most setups | Legacy or LAN-only |
| **TCPMUX** | Supported | Not supported |

## Error Handling

All methods return `error` as the last return value. Use standard Go error handling:

```go
serial, err := zk.SerialNumber()
if err != nil {
    log.Printf("Failed to get serial: %v", err)
    return
}
```

## Helper Functions

```go
// Human-readable attendance state
zkteco.StateName(zkteco.STATE_FINGERPRINT) // "fingerprint"

// Human-readable attendance type
zkteco.TypeName(zkteco.TYPE_CHECK_IN) // "check_in"

// Human-readable event name
zkteco.EventName(zkteco.EF_ATTLOG) // "attendance"
```

## Constants

### Attendance States

| Constant | Value | Name |
|----------|-------|------|
| `STATE_PASSWORD` | 0 | password |
| `STATE_FINGERPRINT` | 1 | fingerprint |
| `STATE_CARD` | 2 | card |

### Attendance Types

| Constant | Value | Name |
|----------|-------|------|
| `TYPE_CHECK_IN` | 0 | check_in |
| `TYPE_CHECK_OUT` | 1 | check_out |
| `TYPE_BREAK_IN` | 2 | break_in |
| `TYPE_BREAK_OUT` | 3 | break_out |
| `TYPE_OVERTIME_IN` | 4 | overtime_in |
| `TYPE_OVERTIME_OUT` | 5 | overtime_out |

### User Roles

| Constant | Value | Description |
|----------|-------|-------------|
| `LEVEL_USER` | 0 | Normal user |
| `LEVEL_ADMIN` | 14 | Administrator |

## Tested Devices

| Device | Protocol | Status |
|--------|----------|--------|
| ZKTeco K40 | TCP | ✅ Fully tested |



## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT License. See [LICENSE](LICENSE) for details.

## Author & Info

**Author:** Mithun Halder  
GitHub: [0mithun](https://github.com/0mithun)  
Email: mithunrptc@gmail.com

For updates, issues, and contributions, visit the [project repository](https://github.com/0mithun/go-zkteco).
