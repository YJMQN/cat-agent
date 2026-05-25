package memory

import (
	"context"
	"sync"
	"time"

	"ai-agent-system/pkg/types"
)

// Store 记忆存储接口
type Store interface {
	// GetSession 获取会话
	GetSession(ctx context.Context, sessionID string) (*types.Session, error)
	// SaveSession 保存会话
	SaveSession(ctx context.Context, session *types.Session) error
	// DeleteSession 删除会话
	DeleteSession(ctx context.Context, sessionID string) error
	// GetUserPreferences 获取用户偏好
	GetUserPreferences(ctx context.Context, userID string) (*types.UserPreferences, error)
	// SaveUserPreferences 保存用户偏好
	SaveUserPreferences(ctx context.Context, prefs *types.UserPreferences) error
}

// MemoryManager 记忆管理器
type MemoryManager struct {
	store           Store
	sessionCache    sync.Map // sessionID -> *types.Session
	prefsCache      sync.Map // userID -> *types.UserPreferences
	maxContextSize  int      // 最大上下文消息数
	maxTokenCount   int      // 最大 token 数
}

// NewMemoryManager 创建记忆管理器
func NewMemoryManager(store Store, maxContextSize, maxTokenCount int) *MemoryManager {
	return &MemoryManager{
		store:          store,
		maxContextSize: maxContextSize,
		maxTokenCount:  maxTokenCount,
	}
}

// GetSession 获取会话（带缓存）
func (m *MemoryManager) GetSession(ctx context.Context, sessionID string) (*types.Session, error) {
	// 尝试从缓存获取
	if cached, ok := m.sessionCache.Load(sessionID); ok {
		return cached.(*types.Session), nil
	}

	// 从存储加载
	session, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 缓存
	m.sessionCache.Store(sessionID, session)
	return session, nil
}

// AddMessage 添加消息到会话
func (m *MemoryManager) AddMessage(ctx context.Context, sessionID string, msg types.Message) error {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		// 创建新会话
		session = &types.Session{
			ID:        sessionID,
			Messages:  []types.Message{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()

	// 上下文窗口管理
	if len(session.Messages) > m.maxContextSize {
		// 保留 system 消息和最新的消息
		session.Messages = m.trimMessages(session.Messages)
	}

	return m.store.SaveSession(ctx, session)
}

// trimMessages 修剪消息列表以适应上下文窗口
func (m *MemoryManager) trimMessages(messages []types.Message) []types.Message {
	if len(messages) <= m.maxContextSize {
		return messages
	}

	var result []types.Message
	var systemMsg *types.Message

	// 保留 system 消息
	for i, msg := range messages {
		if msg.Role == types.RoleSystem {
			systemMsg = &messages[i]
			break
		}
	}

	// 添加 system 消息
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}

	// 从后往前添加消息直到达到限制
	count := 0
	if systemMsg != nil {
		count = 1
	}

	for i := len(messages) - 1; i >= 0 && count < m.maxContextSize; i-- {
		if messages[i].Role == types.RoleSystem {
			continue
		}
		// 在开头插入以保持顺序
		result = append([]types.Message{messages[i]}, result...)
		count++
	}

	return result
}

// GetUserPreferences 获取用户偏好（带缓存）
func (m *MemoryManager) GetUserPreferences(ctx context.Context, userID string) (*types.UserPreferences, error) {
	if cached, ok := m.prefsCache.Load(userID); ok {
		return cached.(*types.UserPreferences), nil
	}

	prefs, err := m.store.GetUserPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	m.prefsCache.Store(userID, prefs)
	return prefs, nil
}

// UpdateUserPreferences 更新用户偏好
func (m *MemoryManager) UpdateUserPreferences(ctx context.Context, prefs *types.UserPreferences) error {
	err := m.store.SaveUserPreferences(ctx, prefs)
	if err != nil {
		return err
	}

	m.prefsCache.Store(prefs.UserID, prefs)
	return nil
}

// ClearSessionCache 清除会话缓存
func (m *MemoryManager) ClearSessionCache(sessionID string) {
	m.sessionCache.Delete(sessionID)
}

// InMemoryStore 内存存储实现（用于测试和演示）
type InMemoryStore struct {
	sessions map[string]*types.Session
	prefs    map[string]*types.UserPreferences
	mu       sync.RWMutex
}

// NewInMemoryStore 创建内存存储
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]*types.Session),
		prefs:    make(map[string]*types.UserPreferences),
	}
}

// GetSession 获取会话
func (s *InMemoryStore) GetSession(ctx context.Context, sessionID string) (*types.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// SaveSession 保存会话
func (s *InMemoryStore) SaveSession(ctx context.Context, session *types.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
	return nil
}

// DeleteSession 删除会话
func (s *InMemoryStore) DeleteSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// GetUserPreferences 获取用户偏好
func (s *InMemoryStore) GetUserPreferences(ctx context.Context, userID string) (*types.UserPreferences, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefs, exists := s.prefs[userID]
	if !exists {
		// 返回默认偏好
		return &types.UserPreferences{UserID: userID}, nil
	}

	return prefs, nil
}

// SaveUserPreferences 保存用户偏好
func (s *InMemoryStore) SaveUserPreferences(ctx context.Context, prefs *types.UserPreferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prefs[prefs.UserID] = prefs
	return nil
}

// 错误定义
var (
	ErrSessionNotFound = &MemoryError{Code: "SESSION_NOT_FOUND", Message: "会话不存在"}
)

// MemoryError 记忆错误
type MemoryError struct {
	Code    string
	Message string
	Err     error
}

func (e *MemoryError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + " - " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *MemoryError) Unwrap() error {
	return e.Err
}
