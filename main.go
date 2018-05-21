package main

import (
	"flag"
	"log"
	"net"
)

type StringArray []string

func (f *StringArray) String() string {
	s := "[ "
	for i:=0; i<len(*f); i++ {
		s += "\""
		s += (*f)[i]
		s += "\", "
	}
	s += "]"
	return s
}

func (f *StringArray) Set(value string) error {
	*f = append(*f, value)
	return nil
}

var suffixs StringArray
var address string

func main() {
	var help bool
	flag.BoolVar(&help, "help", false, "显示此帮助信息")
	flag.StringVar(&address, "bind", ":8080", "此代理程序监听的地址端口")
	flag.Var(&suffixs, "suffix", "允许通过进行透明代理的目标域名结尾, 可进行多次指定 (不指定时, 默认允许所有域名通过)")
	flag.Parse()
	if help {
		flag.PrintDefaults()
		return
	}
	
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("[FATAL] 无法监听网络端口", address, err)
	}
	log.Println("[INFO] 监听", address, "服务已启动")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("[FATAL] 无法接收网络连接:", err)
		}
		go handle(conn)
	}
}


func handle(from net.Conn) {
	h := CreateHandler(from)
	defer h.Close()
	h.Handle()
}
