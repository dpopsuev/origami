package sqlite

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Schema is the top-level YAML definition for a SQLite database.
type Schema struct {
	Version int     `yaml:"version"`
	Tables  []Table `yaml:"tables"`
	Indexes []Index `yaml:"indexes,omitempty"`
}

// Table defines a single SQLite table.
type Table struct {
	Name        string       `yaml:"name"`
	Columns     []Column     `yaml:"columns"`
	ForeignKeys []ForeignKey `yaml:"foreign_keys,omitempty"`
	Unique      [][]string   `yaml:"unique,omitempty"`
}

// Column defines a single column in a table.
type Column struct {
	Name          string `yaml:"name"`
	Type          string `yaml:"type"`
	PrimaryKey    bool   `yaml:"primary_key,omitempty"`
	Autoincrement bool   `yaml:"autoincrement,omitempty"`
	NotNull       bool   `yaml:"not_null,omitempty"`
	Unique        bool   `yaml:"unique,omitempty"`
	Default       string `yaml:"default,omitempty"`
	References    string `yaml:"references,omitempty"`
}

// ForeignKey defines a table-level foreign key constraint.
type ForeignKey struct {
	Columns    []string `yaml:"columns"`
	References string   `yaml:"references"`
}

// Index defines a database index.
type Index struct {
	Name    string   `yaml:"name"`
	Table   string   `yaml:"table"`
	Columns []string `yaml:"columns"`
	Unique  bool     `yaml:"unique,omitempty"`
}

// ParseSchemaFile reads and parses a YAML schema from a file path.
func ParseSchemaFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema file %s: %w", path, err)
	}
	return ParseSchema(data)
}

// ParseSchema parses a YAML schema from bytes.
func ParseSchema(data []byte) (*Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse schema YAML: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks the schema for structural errors.
func (s *Schema) Validate() error {
	if s.Version == 0 {
		return fmt.Errorf("schema version is required")
	}
	tables := make(map[string]bool, len(s.Tables))
	for _, t := range s.Tables {
		if t.Name == "" {
			return fmt.Errorf("table name is required")
		}
		if tables[t.Name] {
			return fmt.Errorf("duplicate table %q", t.Name)
		}
		tables[t.Name] = true
		if len(t.Columns) == 0 {
			return fmt.Errorf("table %q has no columns", t.Name)
		}
		cols := make(map[string]bool, len(t.Columns))
		for _, c := range t.Columns {
			if c.Name == "" {
				return fmt.Errorf("table %q: column name is required", t.Name)
			}
			if cols[c.Name] {
				return fmt.Errorf("table %q: duplicate column %q", t.Name, c.Name)
			}
			cols[c.Name] = true
			if c.Type == "" {
				return fmt.Errorf("table %q: column %q type is required", t.Name, c.Name)
			}
		}
		for _, uc := range t.Unique {
			for _, col := range uc {
				if !cols[col] {
					return fmt.Errorf("table %q: unique constraint references unknown column %q", t.Name, col)
				}
			}
		}
	}
	for _, idx := range s.Indexes {
		if idx.Name == "" {
			return fmt.Errorf("index name is required")
		}
		if !tables[idx.Table] {
			return fmt.Errorf("index %q references unknown table %q", idx.Name, idx.Table)
		}
	}
	return nil
}

// GenerateDDL produces CREATE TABLE and CREATE INDEX statements from the schema.
func (s *Schema) GenerateDDL() string {
	var b strings.Builder
	for i, t := range s.Tables {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(generateTableDDL(t))
	}
	for _, idx := range s.Indexes {
		b.WriteString("\n")
		b.WriteString(generateIndexDDL(idx))
	}
	return b.String()
}

func generateTableDDL(t Table) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", t.Name))

	var lines []string
	for _, c := range t.Columns {
		lines = append(lines, "\t"+generateColumnDDL(c))
	}
	for _, fk := range t.ForeignKeys {
		lines = append(lines, fmt.Sprintf("\tFOREIGN KEY (%s) REFERENCES %s",
			strings.Join(fk.Columns, ", "), fk.References))
	}
	for _, uc := range t.Unique {
		lines = append(lines, fmt.Sprintf("\tUNIQUE(%s)", strings.Join(uc, ", ")))
	}

	b.WriteString(strings.Join(lines, ",\n"))
	b.WriteString("\n);\n")
	return b.String()
}

func generateColumnDDL(c Column) string {
	var parts []string
	parts = append(parts, c.Name, strings.ToUpper(c.Type))
	if c.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if c.Autoincrement {
		parts = append(parts, "AUTOINCREMENT")
	}
	if c.NotNull {
		parts = append(parts, "NOT NULL")
	}
	if c.Unique {
		parts = append(parts, "UNIQUE")
	}
	if c.Default != "" {
		parts = append(parts, "DEFAULT", c.Default)
	}
	if c.References != "" {
		parts = append(parts, "REFERENCES", c.References)
	}
	return strings.Join(parts, " ")
}

func generateIndexDDL(idx Index) string {
	u := ""
	if idx.Unique {
		u = "UNIQUE "
	}
	return fmt.Sprintf("CREATE %sINDEX IF NOT EXISTS %s ON %s(%s);\n",
		u, idx.Name, idx.Table, strings.Join(idx.Columns, ", "))
}
