package main

import (
"bufio"
"flag"
"fmt"
"os"
"strconv"
"strings"
"time"

zkteco "github.com/0mithun/go-zkteco"
)

const (
green  = "\033[32m"
red    = "\033[31m"
yellow = "\033[33m"
cyan   = "\033[36m"
bold   = "\033[1m"
reset  = "\033[0m"
)

var (
host     = flag.String("host", "frp.utso.app", "Device host")
port     = flag.Int("port", 7001, "Device port")
protocol = flag.String("protocol", "tcp", "Protocol")
timeout  = flag.Int("timeout", 30, "Timeout in seconds")
reader   = bufio.NewReader(os.Stdin)
)

func main() {
flag.Parse()

fmt.Println()
fmt.Printf("%s[Password & Realtime Interactive Test]%s\n", bold, reset)
fmt.Printf("Device: %s:%d (%s)\n\n", *host, *port, *protocol)

// PHASE 1: Password tests
fmt.Printf("%s--- PHASE 1: Password Authentication ----%s\n\n", bold, reset)

passwordRound(1, 0, "No password (device should have no password)")

round := 2
for {
fmt.Printf("\n%s> Set a NEW password on your device now.%s\n", yellow, reset)
fmt.Printf("  Enter the password you set (or 'skip'): ")
input := readLine()
if strings.ToLower(strings.TrimSpace(input)) == "skip" {
break
}
pw, err := strconv.Atoi(strings.TrimSpace(input))
if err != nil {
fmt.Printf("  %sInvalid number%s\n", red, reset)
continue
}

passwordRound(round, pw, fmt.Sprintf("password=%d", pw))
round++

wrongPw := pw + 1
fmt.Printf("\n  %sTesting WRONG password (%d)...%s\n", yellow, wrongPw, reset)
passwordRound(round, wrongPw, fmt.Sprintf("WRONG password=%d (expect fail)", wrongPw))
round++

fmt.Printf("\n  Test another password? (yes/no): ")
yn := readLine()
if strings.ToLower(strings.TrimSpace(yn)) != "yes" {
break
}
}

// PHASE 2: Realtime
fmt.Printf("\n%s--- PHASE 2: Realtime Attendance Events ----%s\n\n", bold, reset)
fmt.Printf("  Enter device password (0 for none): ")
pwStr := readLine()
pw, _ := strconv.Atoi(strings.TrimSpace(pwStr))
realtimeTest(pw)
}

func passwordRound(round, password int, desc string) {
fmt.Printf("  %s[Round %d]%s %s\n", cyan, round, reset, desc)

opts := []zkteco.Option{
zkteco.WithProtocol(*protocol),
zkteco.WithTimeout(*timeout),
}
if password > 0 {
opts = append(opts, zkteco.WithPassword(password))
}

zk := zkteco.NewZKTeco(*host, *port, opts...)
err := zk.Connect()
if err != nil {
fmt.Printf("  Connect: %s[FAIL] %s%s\n", red, err, reset)
return
}
fmt.Printf("  Connect: %s[OK]%s\n", green, reset)

serial, err := zk.SerialNumber()
if err != nil {
fmt.Printf("  Serial:  %s[FAIL] %s%s\n", red, err, reset)
} else {
fmt.Printf("  Serial:  %s[OK]%s %s\n", green, reset, serial)
}

info, err := zk.GetMemoryInfo()
if err != nil {
fmt.Printf("  Memory:  %s[FAIL] %s%s\n", red, err, reset)
} else {
fmt.Printf("  Memory:  %s[OK]%s users=%d logs=%d logCap=%d\n",
green, reset, info.UserCount, info.LogCount, info.LogCapacity)
}

zk.Disconnect()
fmt.Printf("  %s[Round %d DONE]%s\n", cyan, round, reset)
}

func realtimeTest(password int) {
opts := []zkteco.Option{
zkteco.WithProtocol(*protocol),
zkteco.WithTimeout(*timeout),
}
if password > 0 {
opts = append(opts, zkteco.WithPassword(password))
}

zk := zkteco.NewZKTeco(*host, *port, opts...)
err := zk.Connect()
if err != nil {
fmt.Printf("  Connect: %s[FAIL] %s%s\n", red, err, reset)
return
}
fmt.Printf("  Connect: %s[OK]%s (password=%d)\n", green, reset, password)

for {
fmt.Printf("\n  %s> Punch your finger on the device now!%s\n", yellow, reset)
fmt.Printf("  Seconds to listen? (default 15, or 'quit'): ")
input := readLine()
input = strings.TrimSpace(input)
if strings.ToLower(input) == "quit" {
break
}

listenSec := 15
if input != "" {
if s, err := strconv.Atoi(input); err == nil && s > 0 {
listenSec = s
}
}

fmt.Printf("  Listening %ds... punch now!\n\n", listenSec)

var events []zkteco.RealTimeEvent
callback := func(event zkteco.RealTimeEvent) {
events = append(events, event)
fmt.Printf("  %s>>> EVENT #%d%s\n", green, len(events), reset)
fmt.Printf("      Type:    %d (%s)\n", event.EventType, event.EventName)
fmt.Printf("      UserID:  %s\n", event.UserID)
fmt.Printf("      State:   %d\n", event.State)
fmt.Printf("      Time:    %s\n", event.Time.Format("2006-01-02 15:04:05"))
fmt.Println()
}

err = zk.GetRealTimeLogs(callback, time.Duration(listenSec)*time.Second)
if err != nil {
fmt.Printf("  %s[ERROR] %s%s\n", red, err, reset)
fmt.Printf("  Reconnecting...\n")
zk.Disconnect()
zk = zkteco.NewZKTeco(*host, *port, opts...)
if err := zk.Connect(); err != nil {
fmt.Printf("  Reconnect: %s[FAIL] %s%s\n", red, err, reset)
return
}
fmt.Printf("  Reconnect: %s[OK]%s\n", green, reset)
continue
}

fmt.Printf("  -- Captured %d event(s) in %ds --\n", len(events), listenSec)
if len(events) == 0 {
fmt.Printf("  %sNo events. Punch during the listen window.%s\n", yellow, reset)
}

fmt.Printf("\n  Listen again? (yes/no): ")
yn := readLine()
if strings.ToLower(strings.TrimSpace(yn)) != "yes" {
break
}
}

zk.Disconnect()
fmt.Printf("\n  %sDone!%s\n\n", green, reset)
}

func readLine() string {
line, _ := reader.ReadString('\n')
return strings.TrimRight(line, "\n\r")
}
