package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type handler struct {
	connFrom net.Conn
	rwFrom  *bufio.ReadWriter
	stat     uint8
	connTo   net.Conn
	rwTo    *bufio.ReadWriter
	size     int64
	keepConn bool
}

const (
	STATUS_REQUEST_TARGET = iota
	STATUS_REQUEST_HEADER
	STATUS_REQUEST_BODY
	STATUS_RESPONSE_STATUS
	STATUS_RESPONSE_HEADER
	STATUS_RESPONSE_BODY
	STATUS_RESPONSE_CHUNK_LENGTH
	STATUS_RESPONSE_CHUNK_BODY
	STATUS_CONNECTION
	STATUS_END
	STATUS_ERROR
)

func CreateHandler(conn net.Conn) *handler {
	return &handler{
		connFrom: conn,
		rwFrom:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		stat:     STATUS_REQUEST_TARGET,
	}
}

func (h *handler) Handle() {
	var err error
PROXY_LOOP:
	for err == nil {
		switch h.stat {
		case STATUS_REQUEST_TARGET:
			err = h.request_target()
		case STATUS_REQUEST_HEADER:
			err = h.request_header()
		case STATUS_REQUEST_BODY:
			err = h.request_body()
		case STATUS_RESPONSE_STATUS:
			err = h.response_status()
		case STATUS_RESPONSE_HEADER:
			err = h.response_header()
		case STATUS_RESPONSE_BODY:
			err = h.response_body()
		case STATUS_RESPONSE_CHUNK_LENGTH:
			err = h.response_chunk_length()
		case STATUS_RESPONSE_CHUNK_BODY:
			err = h.response_chunk_body()
		case STATUS_CONNECTION:
			h.connection()
		case STATUS_END:
			break PROXY_LOOP
		default:
		}
	}
	if err != nil {
		log.Println("[WARN]", err)
	}
}
// 请求首行, 需要解析真实目标, 并建立对应连接(发送初步数据头)
func (h *handler) request_target() error {
	line, err := h.rwFrom.ReadString('\n')
	if err == io.EOF {
		h.stat = STATUS_END
		return nil
	}else if err != nil {
		return err
	}
	
	slices := strings.SplitN(line, " ", 3)
	if slices[1][0:5] != "/http" {
		return errors.New("错误的协议 ("+strings.TrimSpace(line)+")")
	}
	raw, err := url.PathUnescape(slices[1][1:])
	if err != nil {
		return errors.New("地址未转义 ("+strings.TrimSpace(line)+")")
	}
	uri, err := url.Parse(raw)
	if err != nil {
		return errors.New("错误的地址 ("+strings.TrimSpace(line)+")")
	}
	if !hostAllowed(uri.Host) {
		return errors.New("域名被禁止 ("+strings.TrimSpace(line)+")")
	}
	if uri.Scheme == "https" {
		h.connTo, err = tls.Dial("tcp", host2addr(uri.Host, ":443"), nil)
	}else{
		h.connTo, err = net.Dial("tcp", host2addr(uri.Host, ":80"))
	}
	if err != nil {
		h.connTo = nil
		return err
	}
	// 在此处可以认为可以发起请求了, 这里进行一个日志记录
	log.Println("[PROXY] ("+slices[0]+")", uri)
	h.rwTo = bufio.NewReadWriter(bufio.NewReader(h.connTo), bufio.NewWriter(h.connTo))
	_, err = fmt.Fprintf(h.rwTo, "%s %s HTTP/1.1\r\n", slices[0], uri.Path + "?" + uri.RawQuery)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(h.rwTo, "Host: %s\r\n", uri.Host)
	if err != nil {
		return err
	}
	if slices[2] == "HTTP/1.0" {
		h.size = -1
	}else{
		h.size = 0 // 用于标记请求长度
	}
	h.keepConn = false
	h.stat = STATUS_REQUEST_HEADER
	return nil
}
func host2addr(host, suffix string) string {
	if !strings.Contains(host, ":") {
		return host + suffix
	}
	return host
}
func hostAllowed(host string) bool {
	if len(suffixs) == 0 { // 未配置后缀时, 全部允许
		return true
	}
	host = strings.Split(host, ":")[0]
	for _, suffix := range suffixs {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
// 头部信息转发, 需要注意特殊头的处理
func (h *handler) request_header() error {
	line, err := h.rwFrom.ReadString('\n')
	if err != nil {
		
	}else if line == "\r\n" {
		h.stat = STATUS_REQUEST_BODY
		fmt.Fprint(h.rwTo, line) // line 自身包含 "\r"
	} else if strings.HasPrefix(line, "Host:") {
		// Host 头部已发送, 这里忽略即可
	} else if strings.HasPrefix(line, "Content-Length:") {
		h.size, err = strconv.ParseInt(strings.TrimSpace(line[15:]), 10, 64)
		fmt.Fprint(h.rwTo, line)
	} else if strings.HasPrefix(line, "Connection:") {
		// 不支持升级协议, 不发送形如 Connection: Upgrade, ... 头部
		if strings.Contains(line, "Keep-Alive") || strings.Contains(line, "keep-alive") {
			h.keepConn = true
		} else if strings.Contains(line, "Close") || strings.Contains(line, "close") {
			h.keepConn = false
		}
		fmt.Fprint(h.rwTo, line)
	} else if strings.HasPrefix(line, "Upgrade:") ||
		strings.HasPrefix(line, "Sec-WebSocket-") ||
	 	strings.HasPrefix(line, "HTTP2-") {
		// 不支持升级协议, 例如 WebSocket 和 HTTP2
	} else {
		_, err = fmt.Fprint(h.rwTo, line) // line 自身含有 \r\n
	}
	return err
}
// 请求一般两种形式 Content-Length 或 Connection: close 触发
func (h *handler) request_body() error {
	var err error
	if h.size == -1 {
		_, err = io.Copy(h.rwTo, h.rwFrom) // HTTP/1.0 请求
	}else if h.size > 0 {
		_, err = io.CopyN(h.rwTo, h.rwFrom, int64(h.size))
	}
	if err != nil {
		return err
	}
	err = h.rwTo.Flush()
	h.stat = STATUS_RESPONSE_STATUS;
	h.size = 0 // size 将被用于响应长度标记
	return nil
}
func (h *handler) response_status() error {
	line, err := h.rwTo.ReadString('\n')
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(h.rwFrom, line)
	h.stat = STATUS_RESPONSE_HEADER
	return err
}
func (h *handler) response_header() error {
	line, err := h.rwTo.ReadString('\n')
	if err != nil {
	
	}else if line == "\r\n" {
		if h.size >= 0 {
			h.stat = STATUS_RESPONSE_BODY
		}else if h.size == -1 {
			h.stat = STATUS_RESPONSE_CHUNK_LENGTH
		}
		fmt.Fprint(h.rwFrom, line)
	}else if strings.HasPrefix(line, "Content-Length:") {
		h.size, err = strconv.ParseInt(strings.TrimSpace(line[15:]), 10, 64)
		fmt.Fprint(h.rwFrom, line)
	}else if line == "Transfer-Encoding: chunked\r\n" {
		h.size = -1
		fmt.Fprint(h.rwFrom, line)
	}else if strings.HasPrefix(line, "Connection:") {
		if strings.Contains(line, "Close") || strings.Contains(line, "close") {
			h.keepConn = false
		}
		fmt.Fprint(h.rwFrom, line)
	}else{
		fmt.Fprint(h.rwFrom, line)
	}
	return err
}
// 响应体对应 Content-Length 的情况
func (h *handler) response_body() error {
	var err error
	if h.size > 0 {
		_, err = io.CopyN(h.rwFrom, h.rwTo, int64(h.size)) // 注意方向
	}
	if err != nil {
		return err
	}
	err = h.rwFrom.Flush()
	if err != nil {
		return err
	}
	h.stat = STATUS_CONNECTION
	return nil
}
// 响应体处理 chunked 编码流程
func (h *handler) response_chunk_length() error {
	line, err := h.rwTo.ReadString('\n')
	if err != nil {
		return err
	}
	h.size, err = strconv.ParseInt(strings.TrimSpace(line), 16, 32)
	if err != nil {
		return err
	}
	fmt.Fprint(h.rwFrom, line)
	h.stat = STATUS_RESPONSE_CHUNK_BODY
	return nil
}
// 响应体处理 chunked 编码流程
func (h *handler) response_chunk_body() error {
	_, err := io.CopyN(h.rwFrom, h.rwTo, int64(h.size + 2))  // 注意方向
	if err != nil {
		return err
	}
	err = h.rwFrom.Flush()
	if err != nil {
		return err
	}
	if h.size > 0 {
		h.stat = STATUS_RESPONSE_CHUNK_LENGTH
	}else{
		h.stat = STATUS_CONNECTION
	}
	return nil
}
func(h *handler) connection() error {
	if h.keepConn {
		h.stat = STATUS_REQUEST_TARGET
	}else{
		h.stat = STATUS_END
	}
	return nil
}
func(h *handler) Close() {
	if h.connFrom != nil {
		h.connFrom.Close()
	}
	if h.connTo != nil {
		h.connTo.Close()
	}
}
