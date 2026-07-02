package servers

import "time"

// Config 通用服务器配置，包含 gRPC/TCP 服务器的监听地址、超时、消息大小和反射开关等参数
type Config struct {
	Addr             string        // 监听地址 :50051
	Timeout          time.Duration // 连接超时
	MaxRecvMsgSize   int           // 最大接收消息
	MaxSendMsgSize   int           // 最大发送消息
	EnableReflection bool          // 是否开启反射（调试用）
}
