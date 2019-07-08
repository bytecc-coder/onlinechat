package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)
type Client struct {
	//用于发送数据的管道
	C chan string
	//用户名
	Name string
	//网络地址
	Addr string
}
//保存在线用户
var onlineMap map[string]Client
//定义管道
var message = make(chan string)

//处理用户连接
func HandleConn(conn net.Conn)  {
	cliAddr := conn.RemoteAddr().String()

	// 创建一个结构体
	cli := Client{make(chan string),cliAddr,cliAddr}
	onlineMap[cliAddr]=cli

	//新开一个协程，给在线客户端发送信息
	go WriteMessage(cli,conn)
	//广播某个在线
	message <- MakeMessage(cli,"login")
	//提示我是谁
	cli.C <- "i am "+cliAddr
	//定义一个channel判断对方是否主动退出
	isQuit := make(chan bool)
	//定义一个channel判断对方是否超时
	hasData := make(chan bool)
	//新建一个协程，接收用户发送过来的数据
	go func(){
        buf := make([]byte,2048)
        for{
        	n,err := conn.Read(buf)
        	if n==0{
        		isQuit <- true
        		fmt.Println("接收数据失败",err)
				return
			}
        	msg := " say "+string(buf[:n-1])
        	if len(msg)==8&&msg==" say who"{
                conn.Write([]byte("user list:\n"))
                for _,temp:=range onlineMap{
                	conn.Write([]byte(temp.Name+" online \n"))
				}
			}else if len(msg)>=13&&msg[0:11]==" say rename"{
                  newName := strings.Split(msg,"|")[1]
                  cli.Name=newName
				  onlineMap[cliAddr]=cli
				  conn.Write([]byte("update name success!\n"))
			}else{
				//	消息转发
				message<-MakeMessage(cli,msg)
			}
        	//代表有数据
        	hasData <- true
		}
	}()
    for{
    	//通过select检测channel的流动
		select {
    	case <-isQuit:
			message <- MakeMessage(cli," login out")
			delete(onlineMap,cliAddr)
    		return
		case <-hasData:

		case <-time.After(60 * time.Second)://超时处理
			message <- MakeMessage(cli," time out")
			delete(onlineMap,cliAddr)
			conn.Close()
			break
		}
	}
	time.Sleep(time.Millisecond)
}
//设置广播消息
func MakeMessage(cli Client,msg string) (buf string) {
	buf = "["+cli.Addr+"]->"+cli.Name+":"+msg
	return
}

//遍历map发送客户登录信息
func WriteMessage(cli Client,conn net.Conn)  {
	for msg := range cli.C {
		conn.Write([]byte(msg+"\n"))
	}
}
//遍历map,给每个在线成员发送
func Manager()  {
	//给map新建空间
	onlineMap = make(map[string]Client)
	for{
		//没有消息前会阻塞
		msg := <-message
		for _,cli := range onlineMap{
			cli.C <-msg
		}

	}
}


func main()  {
	//	监听
	listenner,err := net.Listen("tcp",":8000")
	if err != nil {
		fmt.Println("err = ",err)
		return
	}
	defer listenner.Close()

	//新开一个协程，转发消息，遍历map,给每个在线成员发送
	go Manager()

	//主协程，循环阻塞等待用户连接
	for{
		conn,err2 := listenner.Accept()
		if err2 != nil{
			fmt.Println("err2 = ",err2)
			continue
		}
		//处理用户连接
		go HandleConn(conn)
     }

}


