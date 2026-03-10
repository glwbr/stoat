package model

import (
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

// dbColumnsToTable converts a database.Column slice to a table.Column slice.
func dbColumnsToTable(cols []database.Column) []table.Column {
	out := make([]table.Column, len(cols))
	for i, c := range cols {
		out[i] = table.Column{
			Key:      c.Key,
			Title:    c.Title,
			Type:     c.Type,
			MinWidth: c.MinWidth,
			Order:    c.Order,
		}
	}
	return out
}

// dbRowsToTable converts a database.Row slice to a table.Row slice.
func dbRowsToTable(rows []database.Row) []table.Row {
	out := make([]table.Row, len(rows))
	for i, r := range rows {
		out[i] = table.Row(r)
	}
	return out
}
