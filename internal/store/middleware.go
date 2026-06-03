package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/ZephyrDeng/openhook/internal/model"
)

func (s *Store) CreateMiddleware(input model.CustomMiddlewareInput) (model.CustomMiddleware, error) {
	if input.Name == "" || input.Code == "" {
		return model.CustomMiddleware{}, fmt.Errorf("name and code are required")
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	id := newID("mw")
	ts := nowMS()
	_, err := s.db.Exec(
		`INSERT INTO middlewares(middleware_id, name, code, enabled, create_at, update_at) VALUES(?, ?, ?, ?, ?, ?)`,
		id, input.Name, input.Code, boolInt(enabled), ts, ts,
	)
	if err != nil {
		return model.CustomMiddleware{}, err
	}
	return s.GetMiddleware(id)
}

func (s *Store) ListMiddlewares() ([]model.CustomMiddleware, error) {
	rows, err := s.db.Query(`SELECT id, middleware_id, name, code, enabled, create_at, update_at FROM middlewares ORDER BY create_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.CustomMiddleware
	for rows.Next() {
		item, err := scanMiddleware(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetMiddleware(middlewareID string) (model.CustomMiddleware, error) {
	row := s.db.QueryRow(`SELECT id, middleware_id, name, code, enabled, create_at, update_at FROM middlewares WHERE middleware_id = ?`, middlewareID)
	item, err := scanMiddleware(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.CustomMiddleware{}, ErrNotFound
	}
	return item, err
}

func (s *Store) GetMiddlewares(ids []string) ([]model.CustomMiddleware, error) {
	items := make([]model.CustomMiddleware, 0, len(ids))
	for _, id := range ids {
		item, err := s.GetMiddleware(id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) UpdateMiddleware(middlewareID string, input model.CustomMiddlewareInput) (model.CustomMiddleware, error) {
	current, err := s.GetMiddleware(middlewareID)
	if err != nil {
		return model.CustomMiddleware{}, err
	}
	if input.Name == "" {
		input.Name = current.Name
	}
	if input.Code == "" {
		input.Code = current.Code
	}
	enabled := current.Enabled
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	_, err = s.db.Exec(`UPDATE middlewares SET name = ?, code = ?, enabled = ?, update_at = ? WHERE middleware_id = ?`, input.Name, input.Code, boolInt(enabled), nowMS(), middlewareID)
	if err != nil {
		return model.CustomMiddleware{}, err
	}
	return s.GetMiddleware(middlewareID)
}

func (s *Store) DeleteMiddleware(middlewareID string) error {
	res, err := s.db.Exec(`DELETE FROM middlewares WHERE middleware_id = ?`, middlewareID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func scanMiddleware(row scanner) (model.CustomMiddleware, error) {
	var item model.CustomMiddleware
	var enabled int
	err := row.Scan(&item.ID, &item.MiddlewareID, &item.Name, &item.Code, &enabled, &item.CreateAt, &item.UpdateAt)
	item.Enabled = scanBool(enabled)
	return item, err
}
