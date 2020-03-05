package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"sync"
)

const (
	getDnsError          = "无法获取dns"
	dnsError             = "dns错误"
	listenError          = "监听端口失败"
	buildConnectionError = "建立连接错误"
	connectionError      = "连接%v失败:%v\n"
	sendError            = "往%v发送数据失败:%v\n"
	acceptError          = "从%v接收数据失败:%v\n"
)

var (
	lock       sync.Mutex
	localPort  = ":9090"  /** -l=localhost:80 本地监听端口 */
	remotePort = ":30443" /** -r=ip:80 指定转发端口 */
	src        = "http://119.29.29.29/d?dn=dns.k3s.work&ip=122.51.56.8"
)

func main()  {
	err := server()
	if err != "" {fmt.Println(err)}
}

func server() string {
	/** 监听本地端口 */
	l, err := net.Listen("tcp", localPort)
	if err != nil {return listenError}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(buildConnectionError)
			continue
		}
		fmt.Printf("send%vto%v\n", conn.RemoteAddr(), conn.LocalAddr())
		go handle(conn)
	}
}

func handle(sconn net.Conn) {
	defer sconn.Close()
	ip, ok := getIP()
	if !ok {
		fmt.Println("httpDns接口出错")
		return
	}
	address    := fmt.Sprint(ip, remotePort)
	dconn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf(connectionError, address, err)
		return
	}
	ExitChan := make(chan bool, 1)
	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		_, err := io.Copy(dconn, sconn)
		fmt.Printf(sendError, address, err)
		ExitChan <- true
	} (sconn, dconn, ExitChan)
	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		_, err := io.Copy(sconn, dconn)
		fmt.Printf(acceptError, address, err)
		ExitChan <- true
	} (sconn, dconn, ExitChan)
	<-ExitChan
	dconn.Close()
}

func getIP() (string, bool) {
	lock.Lock()
	defer lock.Unlock()
	/** 获取ip */
	//return "129.226.124.44", true

	resp, _ := http.Get(src)
	if resp.StatusCode != 200 {
		return getDnsError, false
	}

	bytes, _ := ioutil.ReadAll(resp.Body)
	bLen 	 := len(bytes)

	/** 返回体长度出错 */
	if bLen < 8 {return dnsError, false}
	/** 只有一个IP，直接返回 */
	if bLen < 15 {return string(bytes), true}
	/** 获取第一个IP */
	for i := 0; i < bLen; i++ {
		if bytes[i] == 59 {return string(bytes[:i]), true}
		/** 返回体长度出错 */
		if i > 15 {return dnsError, false}
	}
	return "", false
}

func checkNetWorkStatus() bool {
	cmd := exec.Command("ping", "dns.k3s.work", "-c", "1", "-W", "5")
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}