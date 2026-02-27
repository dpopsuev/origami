package sqlite

import (
	"strings"
	"testing"
)

func TestParseSchema_Valid(t *testing.T) {
	yaml := `
version: 2
tables:
  - name: users
    columns:
      - name: id
        type: integer
        primary_key: true
        autoincrement: true
      - name: email
        type: text
        not_null: true
        unique: true
      - name: name
        type: text
        not_null: true
      - name: status
        type: text
        not_null: true
        default: "'active'"
  - name: posts
    columns:
      - name: id
        type: integer
        primary_key: true
        autoincrement: true
      - name: user_id
        type: integer
        not_null: true
        references: "users(id)"
      - name: title
        type: text
        not_null: true
      - name: body
        type: text
    unique:
      - [user_id, title]
indexes:
  - name: idx_posts_user
    table: posts
    columns: [user_id]
`
	s, err := ParseSchema([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}
	if s.Version != 2 {
		t.Errorf("version = %d, want 2", s.Version)
	}
	if len(s.Tables) != 2 {
		t.Fatalf("tables count = %d, want 2", len(s.Tables))
	}
	if s.Tables[0].Name != "users" {
		t.Errorf("table[0].Name = %q, want users", s.Tables[0].Name)
	}
	if len(s.Tables[0].Columns) != 4 {
		t.Errorf("users columns = %d, want 4", len(s.Tables[0].Columns))
	}
	if len(s.Tables[1].Unique) != 1 {
		t.Errorf("posts unique = %d, want 1", len(s.Tables[1].Unique))
	}
	if len(s.Indexes) != 1 {
		t.Errorf("indexes = %d, want 1", len(s.Indexes))
	}
}

func TestParseSchema_Errors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{"no version", "tables:\n  - name: t\n    columns:\n      - name: c\n        type: text", "version is required"},
		{"no table name", "version: 1\ntables:\n  - columns:\n      - name: c\n        type: text", "table name is required"},
		{"no columns", "version: 1\ntables:\n  - name: t", "has no columns"},
		{"no column name", "version: 1\ntables:\n  - name: t\n    columns:\n      - type: text", "column name is required"},
		{"no column type", "version: 1\ntables:\n  - name: t\n    columns:\n      - name: c", "type is required"},
		{"duplicate table", "version: 1\ntables:\n  - name: t\n    columns:\n      - name: c\n        type: text\n  - name: t\n    columns:\n      - name: c\n        type: text", "duplicate table"},
		{"duplicate column", "version: 1\ntables:\n  - name: t\n    columns:\n      - name: c\n        type: text\n      - name: c\n        type: int", "duplicate column"},
		{"bad index ref", "version: 1\ntables:\n  - name: t\n    columns:\n      - name: c\n        type: text\nindexes:\n  - name: idx\n    table: missing\n    columns: [c]", "unknown table"},
		{"bad unique ref", "version: 1\ntables:\n  - name: t\n    columns:\n      - name: c\n        type: text\n    unique:\n      - [missing]", "unknown column"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSchema([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestGenerateDDL(t *testing.T) {
	s := &Schema{
		Version: 1,
		Tables: []Table{
			{
				Name: "items",
				Columns: []Column{
					{Name: "id", Type: "integer", PrimaryKey: true, Autoincrement: true},
					{Name: "name", Type: "text", NotNull: true},
					{Name: "value", Type: "real", Default: "0.0"},
					{Name: "category_id", Type: "integer", References: "categories(id)"},
				},
				Unique: [][]string{{"name", "category_id"}},
			},
		},
		Indexes: []Index{
			{Name: "idx_items_cat", Table: "items", Columns: []string{"category_id"}},
		},
	}

	ddl := s.GenerateDDL()

	expectations := []string{
		"CREATE TABLE IF NOT EXISTS items",
		"id INTEGER PRIMARY KEY AUTOINCREMENT",
		"name TEXT NOT NULL",
		"value REAL DEFAULT 0.0",
		"category_id INTEGER REFERENCES categories(id)",
		"UNIQUE(name, category_id)",
		"CREATE INDEX IF NOT EXISTS idx_items_cat ON items(category_id)",
	}
	for _, exp := range expectations {
		if !strings.Contains(ddl, exp) {
			t.Errorf("DDL missing %q\nGot:\n%s", exp, ddl)
		}
	}
}

func TestGenerateDDL_ForeignKeys(t *testing.T) {
	s := &Schema{
		Version: 1,
		Tables: []Table{
			{
				Name: "links",
				Columns: []Column{
					{Name: "id", Type: "integer", PrimaryKey: true},
					{Name: "a_id", Type: "integer", NotNull: true},
					{Name: "b_id", Type: "integer", NotNull: true},
				},
				ForeignKeys: []ForeignKey{
					{Columns: []string{"a_id"}, References: "a(id)"},
					{Columns: []string{"b_id"}, References: "b(id)"},
				},
			},
		},
	}

	ddl := s.GenerateDDL()
	if !strings.Contains(ddl, "FOREIGN KEY (a_id) REFERENCES a(id)") {
		t.Errorf("DDL missing FK for a_id\nGot:\n%s", ddl)
	}
	if !strings.Contains(ddl, "FOREIGN KEY (b_id) REFERENCES b(id)") {
		t.Errorf("DDL missing FK for b_id\nGot:\n%s", ddl)
	}
}
