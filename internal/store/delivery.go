package store

import "github.com/ZephyrDeng/openhook/internal/model"

func (s *Store) CreateDelivery(item model.Delivery) error {
	if item.CreateAt == 0 {
		item.CreateAt = nowMS()
	}
	_, err := s.db.Exec(
		`INSERT INTO deliveries(request_id, route_id, template_id, target_url, status_code, success, message, request_body, response_body, create_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.RequestID, item.RouteID, item.TemplateID, item.TargetURL, item.StatusCode, boolInt(item.Success), item.Message, rawOrDefault(item.RequestBody, "{}"), item.ResponseBody, item.CreateAt,
	)
	return err
}

func (s *Store) ListDeliveries(limit, offset int) ([]model.Delivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, request_id, route_id, template_id, target_url, status_code, success, message, request_body, response_body, create_at FROM deliveries ORDER BY create_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Delivery
	for rows.Next() {
		var item model.Delivery
		var success int
		var body string
		if err := rows.Scan(&item.ID, &item.RequestID, &item.RouteID, &item.TemplateID, &item.TargetURL, &item.StatusCode, &success, &item.Message, &body, &item.ResponseBody, &item.CreateAt); err != nil {
			return nil, err
		}
		item.Success = scanBool(success)
		item.RequestBody = []byte(body)
		items = append(items, item)
	}
	return items, rows.Err()
}
