package main

import "net"

type User struct {
	Name string
	Addr string      //用户地址
	Chan chan string //与用户绑定的通道
	Conn net.Conn
}

// 创建一个用户的API
func NewUser(conn net.Conn) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name: userAddr,
		Addr: userAddr,
		Chan: make(chan string),
		Conn: conn,
	}

	// 启动监听当前user通道消息的协程
	go user.ListenMsg()

	return user

}

// 监听当前user通道的方法，一旦有消息就发送给客户端
func (this *User) ListenMsg() {
	for {
		msg := <-this.Chan
		this.Conn.Write([]byte(msg + "\n"))
	}
}
