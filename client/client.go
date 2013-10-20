package main

import (
	"os/exec"
	"os"
	"net"
)

func handleConnection(conn net.Conn) {
	conn.Close()
	println("sending command")
	cmd := exec.Command("shutdown", "-h", "now")
	cmd.Start()
}


func main() {
	ln, err := net.Listen("tcp", ":20010")
	if err != nil {
		println ("awww snap")
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		println ("go a connection")
		handleConnection(conn)
	}
}
