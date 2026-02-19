package main

import (
	"fmt"
	"time"

	zkteco "github.com/0mithun/go-zkteco"
)

func main() {
	fmt.Println("=== Testing New Features ===")

	fmt.Print("1. Connect... ")
	zk := zkteco.NewZKTeco("frp.utso.app", 7001,
		zkteco.WithProtocol("tcp"),
		zkteco.WithTimeout(30),
	)
	if err := zk.Connect(); err != nil {
		fmt.Printf("FAIL: %s\n", err)
		return
	}
	fmt.Println("OK")

	fmt.Print("2. Memory info... ")
	info, err := zk.GetMemoryInfo()
	if err != nil {
		fmt.Printf("FAIL: %s\n", err)
	} else {
		fmt.Printf("OK: admin=%d users=%d userCap=%d logs=%d logCap=%d\n",
			info.AdminCount, info.UserCount, info.UserCapacity, info.LogCount, info.LogCapacity)
	}

	fmt.Print("3. Serial... ")
	serial, err := zk.SerialNumber()
	if err != nil {
		fmt.Printf("FAIL: %s\n", err)
	} else {
		fmt.Printf("OK: %s\n", serial)
	}

	fmt.Print("4. Realtime (3s)... ")
	var ec int
	err = zk.GetRealTimeLogs(func(e zkteco.RealTimeEvent) {
		ec++
		fmt.Printf("\n   Event: %s user=%s", e.EventName, e.UserID)
	}, 3*time.Second)
	if err != nil {
		fmt.Printf("FAIL: %s\n", err)
	} else {
		fmt.Printf("OK: %d events\n", ec)
	}

	fmt.Print("5. Disconnect... ")
	if err := zk.Disconnect(); err != nil {
		fmt.Printf("FAIL: %s\n", err)
	} else {
		fmt.Println("OK")
	}
	fmt.Println("\n=== Done ===")
}
