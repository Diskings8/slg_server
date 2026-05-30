package servers

import "time"

type Config struct {
	Addr             string        // 监听地址 :50051
	Timeout          time.Duration // 连接超时
	MaxRecvMsgSize   int           // 最大接收消息
	MaxSendMsgSize   int           // 最大发送消息
	EnableReflection bool          // 是否开启反射（调试用）
}
