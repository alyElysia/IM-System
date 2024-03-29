package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	// 在线用户的列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex //为用户列表添加的读写锁

	//消息广播的通道
	Message chan string
}

// 创建一个server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:   ip,
		Port: port,

		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}

	return server
}

// 监听消息通道的协程，一旦有消息就发送给全部在线user
func (this *Server) ListenMsger() {
	for {
		msg := <-this.Message //不断尝试读取消息

		// 将msg发送给全部的在线user
		this.mapLock.Lock()
		for _, m := range this.OnlineMap {
			m.Chan <- msg
		}

		this.mapLock.Unlock()
	}
}

// 广播消息的方法
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg //将客户端的通道中的消息发送给用户的通道
}

func (this *Server) Handler(conn net.Conn) {
	//当前链接的业务
	// fmt.Println("链接建立成功")

	user := NewUser(conn, this)

	// 用户上线，将用户加入到OnlineMap中
	user.Online()
	// 监听用户是否活跃的channel
	isLive := make(chan bool)

	// 接收客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf) //n是读取成功后返回的数据的字节数
			if n == 0 {
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err: ", err)
				return
			}

			// 提取用户的消息
			msg := string(buf[:])

			// 用户针对消息进行处理
			user.DoMsg(msg + "\n")

			isLive <- true
		}
	}()

	//当前handler阻塞
	for {
		select {
		case <-isLive: //一定要在计时器上方
			// 当前用户是活跃的，重置计时器，不做任何处理

		// 设置计时器
		case <-time.After(time.Second * 10):
			//超时
			// 将当前用户强制退出
			user.SendMsg("你已被强制退出")

			// 关闭占用资源
			close(user.Chan)

			// 关闭连接
			conn.Close()

			// 退出当前Handler
			return //runtime.Goexit 也可以

		}
	}
}

// 启动服务器的接口
func (this *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err :", err)
		return
	}

	//close listen socket
	defer listener.Close()

	// 启动监听消息的协程
	go this.ListenMsger()

	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err :", err)
			continue
		}

		//do handler
		go this.Handler(conn)
	}
}
