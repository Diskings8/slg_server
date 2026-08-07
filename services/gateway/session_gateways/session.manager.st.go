package session_gateways

// 会话注册表：进服后按 roleID 登记 TCP 会话，供下推 RPC（GatewayService.Stream）按角色定位连接并推送消息给客户端。

import (
	"fmt"
	"sync"

	"server.slg.com/api/protocol/pb/pb_common"
)

// defaultSessionManager 包级会话注册表单例（roleID → Session）
var defaultSessionManager = &SessionManager{
	sessions: make(map[uint64]*Session),
}

// SessionManager 角色会话注册表：进服角色 → 客户端 TCP 会话，支持按 roleID 查询/下推。
//
// 生命周期：
//   - 进服（EnterServer 成功，捕获 roleID）→ Register
//   - 连接断开 → Unregister
//   - 下推 RPC 按 roleID 定位会话（Get）→ 回写客户端
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[uint64]*Session
}

// Register 注册/覆盖角色会话（EnterServer 成功后调用，幂等）
func (m *SessionManager) Register(roleID uint64, s *Session) {
	if roleID < 1 || s == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[roleID] = s
}

// Unregister 注销角色会话（连接断开时调用，幂等）
func (m *SessionManager) Unregister(roleID uint64) {
	if roleID < 1 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, roleID)
}

// Get 按 roleID 查询角色会话（未进服/已断开返回 nil）
func (m *SessionManager) Get(roleID uint64) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[roleID]
}

// PushToRoleID 按 roleID 定位会话并下推消息给客户端（其他节点经 GatewayService.Stream 调用）。
//
// seq 固定为 0（服务端主动下推）；写失败仅记日志（writeNodePacket 内部处理）。
func PushToRoleID(roleID uint64, nodePacket *pb_common.NodePacket) error {
	s := defaultSessionManager.Get(roleID)
	if s == nil {
		return fmt.Errorf("role %d not connected", roleID)
	}
	s.writeNodePacket(0, nodePacket)
	return nil
}
