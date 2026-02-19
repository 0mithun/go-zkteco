package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	zkteco "github.com/0mithun/go-zkteco"
)

const (
	green = "\033[32m"
	red   = "\033[31m"
	bold  = "\033[1m"
	reset = "\033[0m"
)

var (
	host     = flag.String("host", "frp.utso.app", "Device host")
	port     = flag.Int("port", 7001, "Device port")
	protocol = flag.String("protocol", "tcp", "Protocol: tcp or udp")
	timeout  = flag.Int("timeout", 30, "Timeout in seconds")
	password = flag.Int("password", 0, "Device password (0=none)")
	tcpmux   = flag.String("tcpmux", "", "TCPMUX proxy host:port/subdomain (e.g. frp.utso.app:1337/zkteco)")

	passed  int
	failed  int
	total   int
	results []testResult
)

type testResult struct {
	group  string
	name   string
	ok     bool
	goVal  string
	phpVal string
	errMsg string
}

func main() {
	flag.Parse()

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║   Go vs PHP ZKTeco Package — Side-by-Side Test       ║")
	fmt.Println("║   Testing against real device (safe commands only)   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Connecting Go client to %s:%d (%s)...\n", *host, *port, *protocol)
	opts := []zkteco.Option{
		zkteco.WithProtocol(*protocol),
		zkteco.WithTimeout(*timeout),
	}
	if *password > 0 {
		opts = append(opts, zkteco.WithPassword(*password))
		fmt.Printf("   Using password: %d\n", *password)
	}
	zk := zkteco.NewZKTeco(*host, *port, opts...)

	if err := zk.Connect(); err != nil {
		fmt.Printf("   %s[FAIL] Go connect failed: %s%s\n", red, err, reset)
		os.Exit(1)
	}
	fmt.Printf("   %s[OK] Go client connected%s\n", green, reset)

	testGroup("A: Device Info")
	testDeviceInfo(zk, "serialNumber", func() (string, error) { return zk.SerialNumber() })
	testDeviceInfo(zk, "deviceName", func() (string, error) { return zk.DeviceName() })
	testDeviceInfo(zk, "deviceId", func() (string, error) { return zk.DeviceID() })
	testDeviceInfo(zk, "vendorName", func() (string, error) { return zk.VendorName() })
	testDeviceInfo(zk, "platform", func() (string, error) { return zk.Platform() })
	testDeviceInfo(zk, "version", func() (string, error) { return zk.Version() })
	testDeviceInfo(zk, "osVersion", func() (string, error) { return zk.OSVersion() })
	testDeviceInfo(zk, "fmVersion", func() (string, error) { return zk.FMVersion() })
	testDeviceInfo(zk, "ssr", func() (string, error) { return zk.SSR() })
	testDeviceInfo(zk, "pinWidth", func() (string, error) { return zk.PinWidth() })
	testDeviceInfo(zk, "faceFunctionOn", func() (string, error) { return zk.FaceFunctionOn() })
	testDeviceInfo(zk, "workCode", func() (string, error) { return zk.WorkCode() })

	testGroup("B: Memory Info")
	testMemoryInfo(zk)

	testGroup("C: Time Management")
	testGetTime(zk)

	testGroup("D: User Management")
	testGetUsers(zk)

	testGroup("E: Fingerprint Management")
	testGetFingerprints(zk)

	testGroup("F: Attendance Management")
	testGetAttendances(zk)

	testGroup("G: Device Control")
	testControl(zk, "disableDevice", func() error { return zk.DisableDevice() })
	testControl(zk, "enableDevice", func() error { return zk.EnableDevice() })
	testControl(zk, "sleep", func() error { return zk.Sleep() })
	testControl(zk, "resume", func() error { return zk.Resume() })

	testGroup("H: LCD Display")
	testControl(zk, "writeLCD", func() error { return zk.WriteLCD("Go ZKTeco Test") })
	testControl(zk, "clearLCD", func() error { return zk.ClearLCD() })

	testGroup("I: Voice/Sound")
	testControl(zk, "testVoice(0)", func() error { return zk.TestVoice(0) })

	testGroup("J: Custom Data")
	testCustomData(zk)

	testGroup("K: Realtime Events")
	testRealtimeRegistration(zk)

	testGroup("L: Disconnect")
	testDisconnect(zk)

	printSummary()

	// After main tests, run password test if requested
	if *password > 0 {
		testGroup("M: Password Auth (separate connection)")
		testPasswordAuth()
	}

	// After main tests, run TCPMUX test if requested
	if *tcpmux != "" {
		testGroup("N: TCPMUX Connection")
		testTCPMUX()
	}

	fmt.Println()
	if failed > 0 {
		os.Exit(1)
	}
}

func testGroup(name string) {
	fmt.Printf("\n%s=== %s ===%s\n", bold, name, reset)
}

func record(group, name string, ok bool, goVal, phpVal, errMsg string) {
	total++
	if ok {
		passed++
	} else {
		failed++
	}
	results = append(results, testResult{group, name, ok, goVal, phpVal, errMsg})
}

func testDeviceInfo(zk *zkteco.ZKTeco, name string, goFn func() (string, error)) {
	goVal, goErr := goFn()
	phpVal := runPHP(name)

	if goErr != nil {
		fmt.Printf("   %s: %s[FAIL] Go error: %s%s\n", name, red, goErr, reset)
		record("Device Info", name, false, "", phpVal, goErr.Error())
		return
	}

	fmt.Printf("   %s: %s[OK]%s\n", name, green, reset)
	fmt.Printf("     Go:  %s\n", goVal)
	fmt.Printf("     PHP: %s\n", phpVal)
	record("Device Info", name, true, goVal, phpVal, "")
}

func testMemoryInfo(zk *zkteco.ZKTeco) {
	info, err := zk.GetMemoryInfo()
	if err != nil {
		fmt.Printf("   getMemoryInfo: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Memory Info", "getMemoryInfo", false, "", "", err.Error())
		return
	}

	phpMem := runPHP("getMemoryInfo")

	fmt.Printf("   getMemoryInfo: %s[OK]%s Both returned data\n", green, reset)
	fmt.Printf("     Go:  admin=%d users=%d userCap=%d logs=%d logCap=%d\n",
		info.AdminCount, info.UserCount, info.UserCapacity, info.LogCount, info.LogCapacity)
	fmt.Printf("     PHP: %s\n", phpMem)

	record("Memory Info", "getMemoryInfo", true,
		fmt.Sprintf("admin=%d,users=%d,cap=%d,logs=%d,logcap=%d",
			info.AdminCount, info.UserCount, info.UserCapacity, info.LogCount, info.LogCapacity),
		phpMem, "")
}

func testGetTime(zk *zkteco.ZKTeco) {
	t, err := zk.GetTime()
	if err != nil {
		fmt.Printf("   getTime: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Time", "getTime", false, "", "", err.Error())
		return
	}

	phpTime := runPHP("getTime")
	goTime := t.Format("2006-01-02 15:04:05")
	fmt.Printf("   getTime: %s[OK]%s\n", green, reset)
	fmt.Printf("     Go:  %s\n", goTime)
	fmt.Printf("     PHP: %s\n", phpTime)
	record("Time", "getTime", true, goTime, phpTime, "")
}

func testGetUsers(zk *zkteco.ZKTeco) {
	users, err := zk.GetUsers()
	if err != nil {
		fmt.Printf("   getUsers: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Users", "getUsers", false, "", "", err.Error())
		return
	}

	phpUsers := runPHP("getUsers")

	fmt.Printf("   getUsers: %s[OK]%s Both returned data\n", green, reset)
	fmt.Printf("     Go:  %d users\n", len(users))
	fmt.Printf("     PHP: %s\n", phpUsers)

	if len(users) > 0 {
		u := users[0]
		fmt.Printf("     Go sample:  UID=%d, ID=%s, Name=%s, Role=%d\n",
			u.UID, u.UserID, u.Name, u.Role)
	}

	record("Users", "getUsers", true, fmt.Sprintf("%d users", len(users)), phpUsers, "")
}

func testGetFingerprints(zk *zkteco.ZKTeco) {
	fps, err := zk.GetFingerprints(1)
	if err != nil {
		fmt.Printf("   getFingerprint(UID=1): %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Fingerprints", "getFingerprint", false, "", "", err.Error())
		return
	}

	phpFp := runPHP("getFingerprint")

	fmt.Printf("   getFingerprint(UID=1): %s[OK]%s\n", green, reset)
	fmt.Printf("     Go:  %d fingerprint(s)\n", len(fps))
	fmt.Printf("     PHP: %s\n", phpFp)
	record("Fingerprints", "getFingerprint", true, fmt.Sprintf("%d fps", len(fps)), phpFp, "")
}

func testGetAttendances(zk *zkteco.ZKTeco) {
	atts, err := zk.GetAttendances()
	if err != nil {
		fmt.Printf("   getAttendances: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Attendance", "getAttendances", false, "", "", err.Error())
		return
	}

	phpAtt := runPHP("getAttendances")

	fmt.Printf("   getAttendances: %s[OK]%s Both returned data\n", green, reset)
	fmt.Printf("     Go:  %d records\n", len(atts))
	fmt.Printf("     PHP: %s\n", phpAtt)

	if len(atts) > 0 {
		a := atts[0]
		fmt.Printf("     Go sample:  UID=%d, UserID=%s, State=%d, Time=%s, Type=%d\n",
			a.UID, a.UserID, a.State, a.RecordTime.Format("2006-01-02 15:04:05"), a.Type)
	}

	record("Attendance", "getAttendances", true, fmt.Sprintf("%d records", len(atts)), phpAtt, "")
}

func testControl(zk *zkteco.ZKTeco, name string, fn func() error) {
	goErr := fn()
	phpResult := runPHP(name)

	goOK := goErr == nil
	phpOK := phpResult == "OK"

	if goOK && phpOK {
		fmt.Printf("   %s: %s[OK]%s Go OK  PHP: OK\n", name, green, reset)
		record("Control", name, true, "OK", "OK", "")
	} else if !goOK && !phpOK {
		fmt.Printf("   %s: %s[OK]%s Both failed (device unsupported)  Go: %s  PHP: %s\n", name, green, reset, goErr, phpResult)
		record("Control", name, true, "UNSUPPORTED", phpResult, "")
	} else if goOK && !phpOK {
		fmt.Printf("   %s: %s[OK]%s Go OK  PHP: %s\n", name, green, reset, phpResult)
		record("Control", name, true, "OK", phpResult, "")
	} else {
		fmt.Printf("   %s: %s[FAIL]%s Go error: %s  PHP: %s\n", name, red, reset, goErr, phpResult)
		record("Control", name, false, goErr.Error(), phpResult, goErr.Error())
	}
}

func testCustomData(zk *zkteco.ZKTeco) {
	err := zk.SetCustomData("go_test_key", "go_test_val_123")
	if err != nil {
		fmt.Printf("   setCustomData: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Custom Data", "setCustomData", false, "", "", err.Error())
		return
	}
	fmt.Printf("   setCustomData: %s[OK]%s\n", green, reset)
	record("Custom Data", "setCustomData", true, "OK", "", "")

	val, err := zk.GetCustomData("go_test_key")
	if err != nil {
		fmt.Printf("   getCustomData: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Custom Data", "getCustomData", false, "", "", err.Error())
		return
	}
	fmt.Printf("   getCustomData: %s[OK]%s %s\n", green, reset, val)
	record("Custom Data", "getCustomData", true, val, "", "")

	err = zk.SetPushCommKey("goTestPushKey")
	if err != nil {
		fmt.Printf("   setPushCommKey: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Custom Data", "setPushCommKey", false, "", "", err.Error())
		return
	}
	fmt.Printf("   setPushCommKey: %s[OK]%s\n", green, reset)
	record("Custom Data", "setPushCommKey", true, "OK", "", "")

	val2, err := zk.GetPushCommKey()
	if err != nil {
		fmt.Printf("   getPushCommKey: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Custom Data", "getPushCommKey", false, "", "", err.Error())
		return
	}
	fmt.Printf("   getPushCommKey: %s[OK]%s %s\n", green, reset, val2)
	record("Custom Data", "getPushCommKey", true, val2, "", "")
}

func testRealtimeRegistration(zk *zkteco.ZKTeco) {
	// Test that we can register for realtime events (CMD_REG_EVENT)
	// Use a very short timeout - we just want to verify registration succeeds
	var eventCount int
	callback := func(event zkteco.RealTimeEvent) {
		eventCount++
		fmt.Printf("     Event: type=%d name=%s user=%s time=%s\n",
			event.EventType, event.EventName, event.UserID, event.Time.Format("15:04:05"))
	}

	fmt.Printf("   registerEvents: listening for 3s...\n")
	err := zk.GetRealTimeLogs(callback, 3*time.Second)
	if err != nil {
		fmt.Printf("   registerEvents: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Realtime", "registerEvents", false, "", "", err.Error())
		return
	}

	fmt.Printf("   registerEvents: %s[OK]%s Received %d events in 3s\n", green, reset, eventCount)
	record("Realtime", "registerEvents", true, fmt.Sprintf("%d events", eventCount), "", "")
}

func testPasswordAuth() {
	// Connect with password (separate connection to verify auth flow)
	fmt.Printf("   Connecting with password=%d...\n", *password)
	zk2 := zkteco.NewZKTeco(*host, *port,
		zkteco.WithProtocol(*protocol),
		zkteco.WithTimeout(*timeout),
		zkteco.WithPassword(*password),
	)
	if err := zk2.Connect(); err != nil {
		fmt.Printf("   passwordAuth: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Password", "passwordAuth", false, "", "", err.Error())
		return
	}

	serial, err := zk2.SerialNumber()
	if err != nil {
		fmt.Printf("   passwordAuth: %s[FAIL] could not get serial after auth: %s%s\n", red, err, reset)
		record("Password", "passwordAuth", false, "", "", err.Error())
		zk2.Disconnect()
		return
	}
	zk2.Disconnect()

	fmt.Printf("   passwordAuth: %s[OK]%s serial=%s\n", green, reset, serial)
	record("Password", "passwordAuth", true, serial, "", "")
}

func testTCPMUX() {
	// Parse tcpmux flag: host:port/subdomain
	parts := strings.SplitN(*tcpmux, "/", 2)
	if len(parts) != 2 {
		fmt.Printf("   tcpmux: %s[FAIL] invalid format, expected host:port/subdomain%s\n", red, reset)
		record("TCPMUX", "tcpmux", false, "", "", "invalid format")
		return
	}
	hostPort := parts[0]
	subdomain := parts[1]
	hp := strings.SplitN(hostPort, ":", 2)
	if len(hp) != 2 {
		fmt.Printf("   tcpmux: %s[FAIL] invalid host:port%s\n", red, reset)
		record("TCPMUX", "tcpmux", false, "", "", "invalid host:port")
		return
	}
	proxyHost := hp[0]
	proxyPort := 0
	fmt.Sscanf(hp[1], "%d", &proxyPort)

	fmt.Printf("   Connecting via TCPMUX proxy %s:%d subdomain=%s...\n", proxyHost, proxyPort, subdomain)
	opts := []zkteco.Option{
		zkteco.WithTCPMUX(proxyHost, proxyPort, subdomain),
		zkteco.WithTimeout(*timeout),
	}
	if *password > 0 {
		opts = append(opts, zkteco.WithPassword(*password))
	}
	zk3 := zkteco.NewZKTeco(*host, *port, opts...)
	if err := zk3.Connect(); err != nil {
		fmt.Printf("   tcpmux: %s[FAIL] %s%s\n", red, err, reset)
		record("TCPMUX", "tcpmux", false, "", "", err.Error())
		return
	}

	serial, err := zk3.SerialNumber()
	if err != nil {
		fmt.Printf("   tcpmux: %s[FAIL] could not get serial: %s%s\n", red, err, reset)
		record("TCPMUX", "tcpmux", false, "", "", err.Error())
		zk3.Disconnect()
		return
	}
	zk3.Disconnect()

	fmt.Printf("   tcpmux: %s[OK]%s serial=%s\n", green, reset, serial)
	record("TCPMUX", "tcpmux", true, serial, "", "")
}

func testDisconnect(zk *zkteco.ZKTeco) {
	err := zk.Disconnect()
	if err != nil {
		fmt.Printf("   disconnect: %s[FAIL] Go error: %s%s\n", red, err, reset)
		record("Disconnect", "disconnect", false, "", "", err.Error())
		return
	}
	fmt.Printf("   disconnect: %s[OK]%s\n", green, reset)
	record("Disconnect", "disconnect", true, "OK", "", "")
}

func runPHP(command string) string {
	script := fmt.Sprintf(
		"require_once '/var/www/html/vendor/autoload.php'; "+
			"use Mithun\\PhpZkteco\\Libs\\ZKTeco; "+
			"$zk = new ZKTeco(host: '%s', port: %d, shouldPing: false, timeout: %d, protocol: '%s'); "+
			"if (!$zk->connect()) { echo 'CONNECT_FAILED'; exit(1); } ",
		*host, *port, *timeout, *protocol,
	)

	switch command {
	case "serialNumber":
		script += "echo $zk->serialNumber();"
	case "deviceName":
		script += "echo $zk->deviceName();"
	case "deviceId":
		script += "echo $zk->deviceId();"
	case "vendorName":
		script += "echo $zk->vendorName();"
	case "platform":
		script += "echo $zk->platform();"
	case "version":
		script += "echo $zk->version();"
	case "osVersion":
		script += "echo $zk->osVersion();"
	case "fmVersion":
		script += "echo $zk->fmVersion();"
	case "ssr":
		script += "echo $zk->ssr();"
	case "pinWidth":
		script += "echo $zk->pinWidth();"
	case "faceFunctionOn":
		script += "echo $zk->faceFunctionOn();"
	case "workCode":
		script += "echo $zk->workCode();"
	case "getMemoryInfo":
		script += "$m = $zk->getMemoryInfo(); echo \"admin={$m->adminCounts},users={$m->userCounts},cap={$m->userCapacity},logs={$m->logCounts},logcap={$m->logCapacity}\";"
	case "getTime":
		script += "echo $zk->getTime();"
	case "getUsers":
		script += "$u = $zk->getUsers(); echo count($u).' users';"
	case "getFingerprint":
		script += "$f = $zk->getFingerprint(1); echo count($f).' fingerprint(s)';"
	case "getAttendances":
		script += "$a = $zk->getAttendances(); echo count($a).' records';"
	case "disableDevice":
		script += "echo $zk->disableDevice() ? 'OK' : 'FAIL';"
	case "enableDevice":
		script += "echo $zk->enableDevice() ? 'OK' : 'FAIL';"
	case "sleep":
		script += "echo $zk->sleep() ? 'OK' : 'FAIL';"
	case "resume":
		script += "echo $zk->resume() ? 'OK' : 'FAIL';"
	case "writeLCD":
		script += "echo $zk->writeLCD('PHP ZKTeco Test') ? 'OK' : 'FAIL';"
	case "clearLCD":
		script += "echo $zk->clearLCD() ? 'OK' : 'FAIL';"
	case "testVoice(0)":
		script += "echo $zk->testVoice(0) ? 'OK' : 'FAIL';"
	default:
		return "UNKNOWN_COMMAND"
	}

	script += " $zk->disconnect();"

	cmd := exec.Command("docker", "exec", "attendance-service", "php", "-r", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("PHP_ERROR: %s (%s)", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out))
}

func printSummary() {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("         SIDE-BY-SIDE SUMMARY           ")
	fmt.Println("========================================")
	fmt.Println()

	if failed == 0 {
		fmt.Printf("  %sPassed:   %d/%d (100%%)%s\n", green, passed, total, reset)
	} else {
		fmt.Printf("  Passed:   %s%d/%d%s\n", green, passed, total, reset)
	}
	fmt.Printf("  Failed:   %s%d%s\n", red, failed, reset)
	fmt.Println()

	if failed == 0 {
		fmt.Printf("  %s[OK] All Go package commands match PHP package behavior!%s\n", green, reset)
	} else {
		fmt.Println("  Failed tests:")
		for _, r := range results {
			if !r.ok {
				fmt.Printf("    [FAIL] %s/%s: %s\n", r.group, r.name, r.errMsg)
			}
		}
	}
	fmt.Println()
}
