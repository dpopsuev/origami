package store

import (
	_ "embed"

	"github.com/dpopsuev/origami/connectors/sqlite"
)

//go:embed schema.yaml
var schemaData []byte

// LoadSchema parses the embedded schema.yaml.
func LoadSchema() (*sqlite.Schema, error) {
	return sqlite.ParseSchema(schemaData)
}
