package login_tokens

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// tokenTTL 登录票据有效期
const tokenTTL = 24 * time.Hour

// TokenManager 进程内登录票据管理（accountID → token，TTL 24h）
//
// 单节点内存态即可满足当前登录需求；将来多实例共享时需换 redis/etcd 存储。
type TokenManager struct {
	mu     sync.RWMutex
	tokens map[uint64]tokenEntry
}

type tokenEntry struct {
	token  string
	expiry int64 // unix 秒
}

func NewTokenManager() *TokenManager {
	return &TokenManager{tokens: make(map[uint64]tokenEntry)}
}

// defaultManager 包级默认票据管理器（单例，login 启动 / 测试 setup 时 InitManager 设置）
var defaultManager *TokenManager

// InitManager 初始化包级默认票据管理器
func InitManager() { defaultManager = NewTokenManager() }

// Get 获取包级默认票据管理器（须先 InitManager；login_logics 直接访问）
func Get() *TokenManager { return defaultManager }

// Issue 签发新票据并返回（crypto/rand 32B hex；rand.Read 失败概率可忽略）
func (m *TokenManager) Issue(accountID uint64) string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	token := hex.EncodeToString(buf)

	m.mu.Lock()
	m.tokens[accountID] = tokenEntry{token: token, expiry: time.Now().Add(tokenTTL).Unix()}
	m.mu.Unlock()
	return token
}

// Verify 校验票据是否有效；过期项懒清理
func (m *TokenManager) Verify(accountID uint64, token string) bool {
	now := time.Now().Unix()
	m.mu.RLock()
	entry, ok := m.tokens[accountID]
	m.mu.RUnlock()
	if !ok || entry.token != token {
		return false
	}
	if entry.expiry < now {
		m.mu.Lock()
		delete(m.tokens, accountID)
		m.mu.Unlock()
		return false
	}
	return true
}
