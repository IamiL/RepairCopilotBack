package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

type Session struct {
	UserID   string `json:"user_id"`
	Login    string `json:"login"`
	IsAdmin1 bool   `json:"is_admin1"`
	IsAdmin2 bool   `json:"is_admin2"`
}

type SessionRepository struct {
	pool *redis.Pool
}

func NewSessionRepository(redisAddr, redisPassword string) *SessionRepository {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisAddr)
			if err != nil {
				return nil, err
			}
			if redisPassword != "" {
				if _, err := c.Do("AUTH", redisPassword); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) CreateSession(sessionId uuid.UUID, userID uuid.UUID, login string, isAdmin1, isAdmin2 bool) error {
	conn := r.pool.Get()
	defer conn.Close()

	session := Session{
		UserID:   userID.String(),
		Login:    login,
		IsAdmin1: isAdmin1,
		IsAdmin2: isAdmin2,
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	_, err = conn.Do("SETEX", fmt.Sprintf("session:%s", sessionId), 3600*2400, sessionData)
	if err != nil {
		return fmt.Errorf("failed to save session to redis: %w", err)
	}

	return nil
}

func (r *SessionRepository) GetSession(sessionID string) (*Session, error) {
	conn := r.pool.Get()
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("GET", fmt.Sprintf("session:%s", sessionID)))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session from redis: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (r *SessionRepository) DeleteSession(sessionID string) error {
	conn := r.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", fmt.Sprintf("session:%s", sessionID))
	if err != nil {
		return fmt.Errorf("failed to delete session from redis: %w", err)
	}

	return nil
}

func (r *SessionRepository) Close() error {
	return r.pool.Close()
}
