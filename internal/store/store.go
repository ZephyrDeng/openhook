package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ZephyrDeng/openhook/internal/model"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func Migrate(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id TEXT NOT NULL UNIQUE,
			template_key TEXT NOT NULL,
			template_name TEXT NOT NULL,
			content TEXT NOT NULL,
			msg_type TEXT NOT NULL DEFAULT 'markdown',
			script TEXT NOT NULL DEFAULT '',
			async_script TEXT NOT NULL DEFAULT '',
			simulation TEXT NOT NULL DEFAULT '{}',
			create_by TEXT NOT NULL DEFAULT '',
			update_by TEXT NOT NULL DEFAULT '',
			current_owner TEXT NOT NULL DEFAULT '',
			visibility TEXT NOT NULL DEFAULT 'private',
			create_at INTEGER NOT NULL,
			update_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL UNIQUE,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			login TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			create_at INTEGER NOT NULL,
			update_at INTEGER NOT NULL,
			UNIQUE(provider, provider_id)
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			create_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			template_ids TEXT NOT NULL DEFAULT '[]',
			is_cover_all INTEGER NOT NULL DEFAULT 0,
			remark TEXT NOT NULL DEFAULT '',
			expire_at INTEGER NOT NULL DEFAULT 0,
			user_ids TEXT NOT NULL DEFAULT '[]',
			create_by TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT 1,
			create_at INTEGER NOT NULL,
			update_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS routes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			route_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			template_id TEXT NOT NULL,
			owner_user_id TEXT NOT NULL DEFAULT '',
			target_urls TEXT NOT NULL DEFAULT '[]',
			headers TEXT NOT NULL DEFAULT '{}',
			middleware_ids TEXT NOT NULL DEFAULT '[]',
			mode TEXT NOT NULL DEFAULT 'envelope',
			enabled INTEGER NOT NULL DEFAULT 1,
			create_at INTEGER NOT NULL,
			update_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS middlewares (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			middleware_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			code TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			create_at INTEGER NOT NULL,
			update_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS rule_sets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 0,
			domain TEXT NOT NULL DEFAULT '[]',
			platform TEXT NOT NULL DEFAULT '',
			payload TEXT NOT NULL DEFAULT '{}',
			create_by TEXT NOT NULL DEFAULT '',
			update_by TEXT NOT NULL DEFAULT '',
			create_at INTEGER NOT NULL,
			update_at INTEGER NOT NULL,
			UNIQUE(kind, name)
		);`,
		`CREATE TABLE IF NOT EXISTS deliveries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL,
			route_id TEXT NOT NULL DEFAULT '',
			template_id TEXT NOT NULL DEFAULT '',
			target_url TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			success INTEGER NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			request_body TEXT NOT NULL DEFAULT '{}',
			response_body TEXT NOT NULL DEFAULT '',
			create_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_deliveries_request_id ON deliveries(request_id);`,
		`CREATE INDEX IF NOT EXISTS idx_deliveries_created ON deliveries(create_at);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := addColumnIfMissing(db, "routes", "owner_user_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "templates", "visibility", "TEXT NOT NULL DEFAULT 'private'"); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_templates_owner ON templates(current_owner);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_templates_visibility ON templates(visibility);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_routes_owner ON routes(owner_user_id);`); err != nil {
		return err
	}
	return nil
}

func addColumnIfMissing(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func nowMS() int64 {
	return time.Now().UnixMilli()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func scanBool(value int) bool {
	return value != 0
}

func toJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func rawOrDefault(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	return string(raw)
}

func asList(raw string) []string {
	var list []string
	_ = json.Unmarshal([]byte(raw), &list)
	return list
}

func asMap(raw string) map[string]string {
	var value map[string]string
	_ = json.Unmarshal([]byte(raw), &value)
	if value == nil {
		return map[string]string{}
	}
	return value
}

func (s *Store) CreateTemplate(input model.TemplateInput) (model.Template, error) {
	if input.TemplateName == "" || input.Content == "" {
		return model.Template{}, fmt.Errorf("templateName and content are required")
	}
	if input.MsgType == "" {
		input.MsgType = "markdown"
	}
	id := newID("tpl")
	key := newID("key")
	ts := nowMS()
	owner := input.CurrentOwner
	if owner == "" {
		owner = input.CreateBy
	}
	visibility := normalizeTemplateVisibility(input.Visibility)
	_, err := s.db.Exec(
		`INSERT INTO templates(template_id, template_key, template_name, content, msg_type, script, async_script, simulation, create_by, update_by, current_owner, visibility, create_at, update_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, key, input.TemplateName, input.Content, input.MsgType, input.Script, input.AsyncScript, rawOrDefault(input.Simulation, "{}"), input.CreateBy, input.CreateBy, owner, visibility, ts, ts,
	)
	if err != nil {
		return model.Template{}, err
	}
	return s.GetTemplate(id)
}

func (s *Store) ListTemplates(search string, limit, offset int, sortBy, sortOrder string) ([]model.Template, int, error) {
	return s.ListTemplatesForOwner(search, "", limit, offset, sortBy, sortOrder)
}

func (s *Store) ListTemplatesForOwner(search, owner string, limit, offset int, sortBy, sortOrder string) ([]model.Template, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if sortBy != "template_name" && sortBy != "update_at" {
		sortBy = "create_at"
	}
	if strings.ToLower(sortOrder) != "asc" {
		sortOrder = "desc"
	}
	whereParts := []string{}
	args := []any{}
	if owner != "" {
		whereParts = append(whereParts, "(current_owner = ? OR visibility = 'public')")
		args = append(args, owner)
	}
	if search != "" {
		whereParts = append(whereParts, "(template_id = ? OR template_name LIKE ? OR content LIKE ?)")
		like := "%" + search + "%"
		args = append(args, search, like, like)
	}
	where := ""
	if len(whereParts) > 0 {
		where = "WHERE " + strings.Join(whereParts, " AND ")
	}
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM templates `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := fmt.Sprintf(`SELECT id, template_id, template_key, template_name, content, msg_type, script, async_script, simulation, create_by, update_by, current_owner, visibility, create_at, update_at FROM templates %s ORDER BY %s %s LIMIT ? OFFSET ?`, where, sortBy, sortOrder)
	args = append(args, limit, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.Template, 0)
	for rows.Next() {
		item, err := scanTemplate(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Store) GetTemplate(templateID string) (model.Template, error) {
	row := s.db.QueryRow(`SELECT id, template_id, template_key, template_name, content, msg_type, script, async_script, simulation, create_by, update_by, current_owner, visibility, create_at, update_at FROM templates WHERE template_id = ?`, templateID)
	item, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Template{}, ErrNotFound
	}
	return item, err
}

func (s *Store) UpdateTemplate(templateID string, input model.TemplateInput, updateBy string) (model.Template, error) {
	current, err := s.GetTemplate(templateID)
	if err != nil {
		return model.Template{}, err
	}
	if input.TemplateName == "" {
		input.TemplateName = current.TemplateName
	}
	if input.Content == "" {
		input.Content = current.Content
	}
	if input.MsgType == "" {
		input.MsgType = current.MsgType
	}
	if len(input.Simulation) == 0 {
		input.Simulation = current.Simulation
	}
	if input.Visibility == "" {
		input.Visibility = current.Visibility
	}
	_, err = s.db.Exec(
		`UPDATE templates SET template_name = ?, content = ?, msg_type = ?, script = ?, async_script = ?, simulation = ?, visibility = ?, update_by = ?, update_at = ? WHERE template_id = ?`,
		input.TemplateName, input.Content, input.MsgType, input.Script, input.AsyncScript, rawOrDefault(input.Simulation, "{}"), normalizeTemplateVisibility(input.Visibility), updateBy, nowMS(), templateID,
	)
	if err != nil {
		return model.Template{}, err
	}
	return s.GetTemplate(templateID)
}

func (s *Store) DeleteTemplate(templateID string) error {
	res, err := s.db.Exec(`DELETE FROM templates WHERE template_id = ?`, templateID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTemplate(row scanner) (model.Template, error) {
	var item model.Template
	var simulation string
	err := row.Scan(&item.ID, &item.TemplateID, &item.TemplateKey, &item.TemplateName, &item.Content, &item.MsgType, &item.Script, &item.AsyncScript, &simulation, &item.CreateBy, &item.UpdateBy, &item.CurrentOwner, &item.Visibility, &item.CreateAt, &item.UpdateAt)
	item.Simulation = json.RawMessage(simulation)
	return item, err
}

func normalizeTemplateVisibility(value string) string {
	if strings.EqualFold(value, "public") {
		return "public"
	}
	return "private"
}
