package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ZephyrDeng/openhook/internal/model"
)

type UserInput struct {
	Provider   string
	ProviderID string
	Login      string
	Name       string
	AvatarURL  string
}

func (s *Store) UpsertUser(input UserInput) (model.User, error) {
	if input.Provider == "" || input.ProviderID == "" || input.Login == "" {
		return model.User{}, fmt.Errorf("provider, providerID and login are required")
	}
	ts := nowMS()
	userID := newID("usr")
	_, err := s.db.Exec(
		`INSERT INTO users(user_id, provider, provider_id, login, name, avatar_url, create_at, update_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, provider_id) DO UPDATE SET
			login = excluded.login,
			name = excluded.name,
			avatar_url = excluded.avatar_url,
			update_at = excluded.update_at`,
		userID, input.Provider, input.ProviderID, input.Login, input.Name, input.AvatarURL, ts, ts,
	)
	if err != nil {
		return model.User{}, err
	}
	return s.GetUserByProvider(input.Provider, input.ProviderID)
}

func (s *Store) GetUser(userID string) (model.User, error) {
	row := s.db.QueryRow(`SELECT id, user_id, provider, provider_id, login, name, avatar_url, create_at, update_at FROM users WHERE user_id = ?`, userID)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) GetUserByProvider(provider, providerID string) (model.User, error) {
	row := s.db.QueryRow(`SELECT id, user_id, provider, provider_id, login, name, avatar_url, create_at, update_at FROM users WHERE provider = ? AND provider_id = ?`, provider, providerID)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) CreateSession(userID string, ttl time.Duration) (model.Session, error) {
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	user, err := s.GetUser(userID)
	if err != nil {
		return model.Session{}, err
	}
	token := newID("ses")
	ts := nowMS()
	expiresAt := time.Now().Add(ttl).UnixMilli()
	_, err = s.db.Exec(`INSERT INTO sessions(token, user_id, expires_at, create_at) VALUES(?, ?, ?, ?)`, token, user.UserID, expiresAt, ts)
	if err != nil {
		return model.Session{}, err
	}
	return model.Session{Token: token, User: user, ExpiresAt: expiresAt, CreateAt: ts}, nil
}

func (s *Store) GetSession(token string) (model.Session, error) {
	if token == "" {
		return model.Session{}, ErrNotFound
	}
	var userID string
	var expiresAt, createAt int64
	err := s.db.QueryRow(`SELECT user_id, expires_at, create_at FROM sessions WHERE token = ?`, token).Scan(&userID, &expiresAt, &createAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Session{}, ErrNotFound
	}
	if err != nil {
		return model.Session{}, err
	}
	if expiresAt <= time.Now().UnixMilli() {
		_ = s.DeleteSession(token)
		return model.Session{}, ErrNotFound
	}
	user, err := s.GetUser(userID)
	if err != nil {
		return model.Session{}, err
	}
	return model.Session{Token: token, User: user, ExpiresAt: expiresAt, CreateAt: createAt}, nil
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func scanUser(row scanner) (model.User, error) {
	var user model.User
	err := row.Scan(&user.ID, &user.UserID, &user.Provider, &user.ProviderID, &user.Login, &user.Name, &user.AvatarURL, &user.CreateAt, &user.UpdateAt)
	return user, err
}
