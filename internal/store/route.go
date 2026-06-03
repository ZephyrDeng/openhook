package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/ZephyrDeng/openhook/internal/model"
)

func (s *Store) CreateRoute(input model.RouteInput) (model.Route, error) {
	if input.Name == "" || input.TemplateID == "" || len(input.TargetURLs) == 0 {
		return model.Route{}, fmt.Errorf("name, templateId and targetUrls are required")
	}
	if input.Mode == "" {
		input.Mode = "envelope"
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	id := newID("rt")
	ts := nowMS()
	_, err := s.db.Exec(
		`INSERT INTO routes(route_id, name, template_id, target_urls, headers, middleware_ids, mode, enabled, create_at, update_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, input.Name, input.TemplateID, toJSON(input.TargetURLs), toJSON(input.Headers), toJSON(input.MiddlewareIDs), input.Mode, boolInt(enabled), ts, ts,
	)
	if err != nil {
		return model.Route{}, err
	}
	return s.GetRoute(id)
}

func (s *Store) ListRoutes() ([]model.Route, error) {
	rows, err := s.db.Query(`SELECT id, route_id, name, template_id, target_urls, headers, middleware_ids, mode, enabled, create_at, update_at FROM routes ORDER BY create_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Route
	for rows.Next() {
		item, err := scanRoute(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetRoute(routeID string) (model.Route, error) {
	row := s.db.QueryRow(`SELECT id, route_id, name, template_id, target_urls, headers, middleware_ids, mode, enabled, create_at, update_at FROM routes WHERE route_id = ?`, routeID)
	item, err := scanRoute(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Route{}, ErrNotFound
	}
	return item, err
}

func (s *Store) UpdateRoute(routeID string, input model.RouteInput) (model.Route, error) {
	current, err := s.GetRoute(routeID)
	if err != nil {
		return model.Route{}, err
	}
	if input.Name == "" {
		input.Name = current.Name
	}
	if input.TemplateID == "" {
		input.TemplateID = current.TemplateID
	}
	if input.TargetURLs == nil {
		input.TargetURLs = current.TargetURLs
	}
	if input.Headers == nil {
		input.Headers = current.Headers
	}
	if input.MiddlewareIDs == nil {
		input.MiddlewareIDs = current.MiddlewareIDs
	}
	if input.Mode == "" {
		input.Mode = current.Mode
	}
	enabled := current.Enabled
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	_, err = s.db.Exec(
		`UPDATE routes SET name = ?, template_id = ?, target_urls = ?, headers = ?, middleware_ids = ?, mode = ?, enabled = ?, update_at = ? WHERE route_id = ?`,
		input.Name, input.TemplateID, toJSON(input.TargetURLs), toJSON(input.Headers), toJSON(input.MiddlewareIDs), input.Mode, boolInt(enabled), nowMS(), routeID,
	)
	if err != nil {
		return model.Route{}, err
	}
	return s.GetRoute(routeID)
}

func (s *Store) DeleteRoute(routeID string) error {
	res, err := s.db.Exec(`DELETE FROM routes WHERE route_id = ?`, routeID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func scanRoute(row scanner) (model.Route, error) {
	var item model.Route
	var targetURLs, headers, middlewareIDs string
	var enabled int
	err := row.Scan(&item.ID, &item.RouteID, &item.Name, &item.TemplateID, &targetURLs, &headers, &middlewareIDs, &item.Mode, &enabled, &item.CreateAt, &item.UpdateAt)
	item.TargetURLs = asList(targetURLs)
	item.Headers = asMap(headers)
	item.MiddlewareIDs = asList(middlewareIDs)
	item.Enabled = scanBool(enabled)
	return item, err
}
