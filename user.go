package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string      //用户地址
	Chan chan string //与用户绑定的通道
	Conn net.Conn

	server *Server //指明用户所属的服务器
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		Chan:   make(chan string),
		Conn:   conn,
		server: server,
	}

	// 启动监听当前user通道消息的协程
	go user.ListenMsg()

	return user

}

// 用户的上线业务
func (this *User) Online() {
	// 用户上线，将用户加入到OnlineMap中
	this.server.mapLock.Lock() //在协程中，加锁防止资源竞争
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()

	// 广播当前用户上线的消息
	this.server.BroadCast(this, "already online~")
}

// 用户的下线业务
func (this *User) Offline() {
	// 用户下线，将用户从OnlineMap中删除
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()

	// 广播当前用户下线的消息
	this.server.BroadCast(this, "already offline~")
}

// 给当前用户对应的客户端发送消息
func (this *User) SendMsg(msg string) {
	this.Conn.Write([]byte(msg))
}

// 处理消息业务
func (this *User) DoMsg(msg string) {
	if msg == "who" {
		// 查询当前在线的用户
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "(" + user.Addr + ":" + user.Name + "),在线...\n"
			this.SendMsg(onlineMsg)
		}

		this.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" { //约定重命名格式：rename|newName
		newName := strings.Split(msg, "|")[1] //含义是将msg以 | 为分隔符分割成一个字符串数组，然后取下标为1的元素

		// 判断name是否存在
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前用户名已被使用\n")
		} else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()

			this.Name = newName
			this.SendMsg("您已成功更新用户名：" + this.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" { //约定发送私聊信息的格式：to|发送对象的用户名|内容

		// 1.获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.SendMsg("格式错误，请重新输入~\n")
			return
		}

		// 2.根据用户名得到对方的User对象
		remoteUser, ok := this.server.OnlineMap[remoteName]
		if !ok {
			this.SendMsg("该用户不存在~\n")
			return
		}

		// 3.获取消息内容，通过对方的User对象将消息内容发送过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.SendMsg("内容为空，请重新编辑~\n")
			return
		}
		remoteUser.SendMsg(this.Name + ":" + content)
	} else {
		this.server.BroadCast(this, msg)
	}

}

// 监听当前user通道的方法，一旦有消息就发送给客户端
func (this *User) ListenMsg() {
	for {
		msg := <-this.Chan
		this.Conn.Write([]byte(msg + "\n"))
	}
}
