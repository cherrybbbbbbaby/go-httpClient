package main

import (
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "www.baidu.com")
	if err != nil {
		println(err.Error())
	}
	println(conn.RemoteAddr().String())
	conn.Write([]byte("1"))
}
