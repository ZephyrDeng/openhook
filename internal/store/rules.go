package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ZephyrDeng/openhook/internal/model"
)

func (s *Store) CreateRuleSet(kind string, input model.RuleSetInput) (model.RuleSet, error) {
	if kind == "" || input.Name == "" {
		return model.RuleSet{}, fmt.Errorf("kind and name are required")
	}
	if len(input.Payload) == 0 {
		input.Payload = json.RawMessage("{}")
	}
	ts := nowMS()
	_, err := s.db.Exec(
		`INSERT INTO rule_sets(kind, name, status, domain, platform, payload, create_by, update_by, create_at, update_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		kind, input.Name, boolInt(input.Status), toJSON(input.Domain), input.Platform, string(input.Payload), input.CreateBy, input.CreateBy, ts, ts,
	)
	if err != nil {
		return model.RuleSet{}, err
	}
	return s.GetRuleSetByName(kind, input.Name)
}

func (s *Store) ListRuleSets(kind string, query map[string][]string) ([]model.RuleSet, error) {
	where := []string{"kind = ?"}
	args := []any{kind}
	if name := first(query, "name"); name != "" {
		where = append(where, "name LIKE ?")
		args = append(args, "%"+name+"%")
	}
	if platform := first(query, "platform"); platform != "" {
		where = append(where, "platform = ?")
		args = append(args, platform)
	}
	stmt := `SELECT id, kind, name, status, domain, platform, payload, create_by, update_by, create_at, update_at FROM rule_sets WHERE ` + strings.Join(where, " AND ") + ` ORDER BY create_at DESC`
	rows, err := s.db.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.RuleSet
	for rows.Next() {
		item, err := scanRuleSet(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return filterRuleSetsByDomain(items, query["domain"]), rows.Err()
}

func (s *Store) GetActiveRuleSet(kind string, query map[string][]string) (model.RuleSet, error) {
	items, err := s.ListRuleSets(kind, query)
	if err != nil {
		return model.RuleSet{}, err
	}
	for _, item := range items {
		if item.Status {
			return item, nil
		}
	}
	return model.RuleSet{}, ErrNotFound
}

func (s *Store) GetRuleSetByName(kind, name string) (model.RuleSet, error) {
	row := s.db.QueryRow(`SELECT id, kind, name, status, domain, platform, payload, create_by, update_by, create_at, update_at FROM rule_sets WHERE kind = ? AND name = ?`, kind, name)
	item, err := scanRuleSet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.RuleSet{}, ErrNotFound
	}
	return item, err
}

func (s *Store) UpdateRuleSet(kind, id string, input model.RuleSetInput) (model.RuleSet, error) {
	current, err := s.GetRuleSet(kind, id)
	if err != nil {
		return model.RuleSet{}, err
	}
	if input.Name == "" {
		input.Name = current.Name
	}
	if input.Domain == nil {
		input.Domain = current.Domain
	}
	if input.Platform == "" {
		input.Platform = current.Platform
	}
	if len(input.Payload) == 0 {
		input.Payload = current.Payload
	}
	_, err = s.db.Exec(
		`UPDATE rule_sets SET name = ?, status = ?, domain = ?, platform = ?, payload = ?, update_by = ?, update_at = ? WHERE kind = ? AND CAST(id AS TEXT) = ?`,
		input.Name, boolInt(input.Status), toJSON(input.Domain), input.Platform, string(input.Payload), input.CreateBy, nowMS(), kind, id,
	)
	if err != nil {
		return model.RuleSet{}, err
	}
	return s.GetRuleSet(kind, id)
}

func (s *Store) GetRuleSet(kind, id string) (model.RuleSet, error) {
	row := s.db.QueryRow(`SELECT id, kind, name, status, domain, platform, payload, create_by, update_by, create_at, update_at FROM rule_sets WHERE kind = ? AND CAST(id AS TEXT) = ?`, kind, id)
	item, err := scanRuleSet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.RuleSet{}, ErrNotFound
	}
	return item, err
}

func (s *Store) DeleteRuleSet(kind, id string) error {
	res, err := s.db.Exec(`DELETE FROM rule_sets WHERE kind = ? AND CAST(id AS TEXT) = ?`, kind, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func scanRuleSet(row scanner) (model.RuleSet, error) {
	var item model.RuleSet
	var status int
	var domain, payload string
	err := row.Scan(&item.ID, &item.Kind, &item.Name, &status, &domain, &item.Platform, &payload, &item.CreateBy, &item.UpdateBy, &item.CreateAt, &item.UpdateAt)
	item.Status = scanBool(status)
	item.Domain = asList(domain)
	item.Payload = json.RawMessage(payload)
	return item, err
}

func first(values map[string][]string, key string) string {
	list := values[key]
	if len(list) == 0 {
		return ""
	}
	return list[0]
}

func filterRuleSetsByDomain(items []model.RuleSet, domains []string) []model.RuleSet {
	if len(domains) == 0 {
		return items
	}
	want := map[string]bool{}
	for _, domain := range domains {
		for _, part := range strings.Split(domain, ",") {
			if part != "" {
				want[part] = true
			}
		}
	}
	if len(want) == 0 {
		return items
	}
	var filtered []model.RuleSet
	for _, item := range items {
		for _, domain := range item.Domain {
			if want[domain] {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}
