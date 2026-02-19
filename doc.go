// Package zkteco provides a Go client for ZKTeco biometric attendance devices.
//
// It implements the ZKTeco proprietary binary protocol over TCP and UDP,
// compatible with the same devices supported by the 0mithun/php-zkteco PHP package.
//
// Usage:
//
//	zk := zkteco.NewZKTeco("192.168.1.201", 4370,
//		zkteco.WithProtocol("tcp"),
//		zkteco.WithTimeout(25),
//	)
//	if err := zk.Connect(); err != nil {
//		log.Fatal(err)
//	}
//	defer zk.Disconnect()
//
//	serial, _ := zk.SerialNumber()
//	fmt.Println("Serial:", serial)
package zkteco
