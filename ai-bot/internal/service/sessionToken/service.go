package sessionTokenService

import (
	"github.com/google/uuid"
	"log/slog"
)

type SessionTokenService struct {
	log             *slog.Logger
	sessionProvider SessionProvider
	sessionSaver    SessionSaver
}

type SessionProvider interface {
	UserId(sessionID uuid.UUID) (uuid.UUID, error)
}

type SessionSaver interface {
	New(userID uuid.UUID, sessionToken uuid.UUID) error
}

func NewSessionTokenService(log *slog.Logger, sessionProvider SessionProvider, sessionSaver SessionSaver) *SessionTokenService {
	return &SessionTokenService{log, sessionProvider, sessionSaver}
}

func (s *SessionTokenService) SessionToken(userId uuid.UUID) (uuid.UUID, error) {
	sessionToken, err := uuid.NewUUID()
	if err != nil {
		return uuid.Nil, err
	}

	err = s.sessionSaver.New(userId, sessionToken)
	if err != nil {
		return uuid.Nil, err
	}

	return sessionToken, nil

}

func (s *SessionTokenService) UserID(sessionToken uuid.UUID) (uuid.UUID, error) {
	uid, err := s.sessionProvider.UserId(sessionToken)
	if err != nil {
		return uuid.Nil, err
	}

	return uid, nil
}
