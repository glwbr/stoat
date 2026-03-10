package model

import (
	"reflect"
	"testing"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/components/table"
)

func TestDBColumnsToTable(t *testing.T) {
	tests := []struct {
		name string
		in   []database.Column
		want []table.Column
	}{
		{
			name: "empty_slice",
			in:   []database.Column{},
			want: []table.Column{},
		},
		{
			name: "single_column_maps_all_fields",
			in: []database.Column{
				{
					Key:      "id",
					Title:    "ID",
					Type:     "INTEGER",
					MinWidth: 6,
					Order:    1,
				},
			},
			want: []table.Column{
				{
					Key:      "id",
					Title:    "ID",
					Type:     "INTEGER",
					MinWidth: 6,
					Order:    1,
				},
			},
		},
		{
			name: "multiple_columns_preserve_order",
			in: []database.Column{
				{Key: "name", Title: "Name", Type: "TEXT", MinWidth: 12, Order: 2},
				{Key: "email", Title: "Email", Type: "TEXT", MinWidth: 16, Order: 3},
			},
			want: []table.Column{
				{Key: "name", Title: "Name", Type: "TEXT", MinWidth: 12, Order: 2},
				{Key: "email", Title: "Email", Type: "TEXT", MinWidth: 16, Order: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dbColumnsToTable(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("dbColumnsToTable() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDBRowsToTable(t *testing.T) {
	tests := []struct {
		name string
		in   []database.Row
		want []table.Row
	}{
		{
			name: "empty_slice",
			in:   []database.Row{},
			want: []table.Row{},
		},
		{
			name: "single_row_maps_fields",
			in: []database.Row{
				{"id": "1", "name": "alice"},
			},
			want: []table.Row{
				{"id": "1", "name": "alice"},
			},
		},
		{
			name: "multiple_rows_preserve_order",
			in: []database.Row{
				{"id": "1", "name": "alice"},
				{"id": "2", "name": "bob"},
			},
			want: []table.Row{
				{"id": "1", "name": "alice"},
				{"id": "2", "name": "bob"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dbRowsToTable(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("dbRowsToTable() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
