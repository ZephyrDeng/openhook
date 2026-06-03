package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ZephyrDeng/openhook/internal/model"
)

func (s *Store) CreateToken(input model.TokenInput) (model.Token, error) {
	if input.Name == "" {
		return model.Token{}, fmt.Errorf("name is required")
	}
	if len(input.TemplateIDs) == 0 && !input.IsCoverAll {
		return model.Token{}, fmt.Errorf("templateIds is required unless isCoverAll is true")
	}
	token := newID("tok")
	ts := nowMS()
	_, err := s.db.Exec(
		`INSERT INTO tokens(token, name, template_ids, is_cover_all, remark, expire_at, user_ids, create_by, status, create_at, update_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		token, input.Name, toJSON(input.TemplateIDs), boolInt(input.IsCoverAll), input.Remark, input.ExpireAt, toJSON(input.UserIDs), input.CreateBy, model.TokenEnabled, ts, ts,
	)
	if err != nil {
		return model.Token{}, err
	}
	return s.GetToken(token)
}

func (s *Store) ListTokens() ([]model.Token, error) {
	rows, err := s.db.Query(`SELECT id, token, name, template_ids, is_cover_all, remark, expire_at, user_ids, create_by, status, create_at, update_at FROM tokens ORDER BY create_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Token
	for rows.Next() {
		item, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		item = normalizeTokenStatus(item)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetToken(idOrToken string) (model.Token, error) {
	row := s.db.QueryRow(`SELECT id, token, name, template_ids, is_cover_all, remark, expire_at, user_ids, create_by, status, create_at, update_at FROM tokens WHERE token = ? OR CAST(id AS TEXT) = ?`, idOrToken, idOrToken)
	item, err := scanToken(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Token{}, ErrNotFound
	}
	if err != nil {
		return model.Token{}, err
	}
	item = normalizeTokenStatus(item)
	if item.Status == model.TokenExpired {
		_, _ = s.db.Exec(`UPDATE tokens SET status = ?, update_at = ? WHERE token = ? AND status = ?`, model.TokenExpired, nowMS(), item.Token, model.TokenEnabled)
	}
	return item, nil
}

func (s *Store) UpdateToken(token string, input model.TokenInput) (model.Token, error) {
	current, err := s.GetToken(token)
	if err != nil {
		return model.Token{}, err
	}
	if current.Status == model.TokenDeleted {
		return model.Token{}, ErrNotFound
	}
	if input.Name == "" {
		input.Name = current.Name
	}
	if input.TemplateIDs == nil {
		input.TemplateIDs = current.TemplateIDs
	}
	if input.UserIDs == nil {
		input.UserIDs = current.UserIDs
	}
	status := model.TokenEnabled
	if input.ExpireAt > 0 && input.ExpireAt < time.Now().UnixMilli() {
		status = model.TokenExpired
	}
	_, err = s.db.Exec(
		`UPDATE tokens SET name = ?, template_ids = ?, is_cover_all = ?, remark = ?, expire_at = ?, user_ids = ?, status = ?, update_at = ? WHERE token = ?`,
		input.Name, toJSON(input.TemplateIDs), boolInt(input.IsCoverAll), input.Remark, input.ExpireAt, toJSON(input.UserIDs), status, nowMS(), token,
	)
	if err != nil {
		return model.Token{}, err
	}
	return s.GetToken(token)
}

func (s *Store) DeleteToken(token string) error {
	res, err := s.db.Exec(`UPDATE tokens SET status = ?, update_at = ? WHERE token = ?`, model.TokenDeleted, nowMS(), token)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) TokenCanEditTemplate(tokenValue, templateID string) (bool, error) {
	token, err := s.GetToken(tokenValue)
	if err != nil {
		return false, err
	}
	if token.Status != model.TokenEnabled {
		return false, nil
	}
	if token.IsCoverAll {
		return true, nil
	}
	for _, id := range token.TemplateIDs {
		if id == templateID {
			return true, nil
		}
	}
	return false, nil
}

func normalizeTokenStatus(item model.Token) model.Token {
	if item.Status == model.TokenEnabled && item.ExpireAt > 0 && item.ExpireAt < time.Now().UnixMilli() {
		item.Status = model.TokenExpired
	}
	return item
}

func scanToken(row scanner) (model.Token, error) {
	var item model.Token
	var templateIDs, userIDs string
	var isCoverAll int
	var status int
	err := row.Scan(&item.ID, &item.Token, &item.Name, &templateIDs, &isCoverAll, &item.Remark, &item.ExpireAt, &userIDs, &item.CreateBy, &status, &item.CreateAt, &item.UpdateAt)
	item.TemplateIDs = asList(templateIDs)
	item.UserIDs = asList(userIDs)
	item.IsCoverAll = scanBool(isCoverAll)
	item.Status = model.TokenStatus(status)
	return item, err
}
