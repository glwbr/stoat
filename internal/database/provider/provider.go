package provider

import (
	"strings"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/database/sqlite"
)

// FromConfig creates a new database connection from the given configuration.
func FromConfig(config database.Config) (database.Connection, error) {
	switch config.DBMS {
	case database.DBMSSQLite:
		return sqlite.NewConnection(config)
	default:
		if strings.TrimSpace(string(config.DBMS)) == "" {
			return nil, database.ErrInvalidConfig
		}
		return nil, database.ErrNotSupported
	}
}
