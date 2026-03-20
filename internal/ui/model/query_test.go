package model

import (
	"strings"
	"testing"
)

func TestFormatSQLValue(t *testing.T) {
	tests := []struct {
		name    string
		colType string
		value   string
		want    string
	}{
		{
			name:    "empty_value_returns_NULL",
			colType: "text",
			value:   "",
			want:    "NULL",
		},
		{
			name:    "whitespace_only_returns_NULL",
			colType: "text",
			value:   "   \t  ",
			want:    "NULL",
		},
		{
			name:    "integer_type_valid_int_unquoted",
			colType: "integer",
			value:   "42",
			want:    "42",
		},
		{
			name:    "integer_type_negative_unquoted",
			colType: "INT",
			value:   "-1",
			want:    "-1",
		},
		{
			name:    "integer_type_non_numeric_quoted",
			colType: "integer",
			value:   "abc",
			want:    "'abc'",
		},
		{
			name:    "numeric_type_valid_int_unquoted",
			colType: "NUMERIC",
			value:   "0",
			want:    "0",
		},
		{
			name:    "real_type_valid_float_unquoted",
			colType: "real",
			value:   "3.14",
			want:    "3.14",
		},
		{
			name:    "float_type_valid_unquoted",
			colType: "FLOAT",
			value:   "1e-2",
			want:    "1e-2", // original value returned when parse succeeds
		},
		{
			name:    "real_type_non_numeric_quoted",
			colType: "REAL",
			value:   "nope",
			want:    "'nope'",
		},
		{
			name:    "text_type_quoted",
			colType: "text",
			value:   "hello",
			want:    "'hello'",
		},
		{
			name:    "text_type_single_quote_escaped",
			colType: "TEXT",
			value:   "O'Brien",
			want:    "'O''Brien'",
		},
		{
			name:    "unknown_type_quoted",
			colType: "blob",
			value:   "x",
			want:    "'x'",
		},
		{
			name:    "value_trimmed",
			colType: "text",
			value:   "  hello  ",
			want:    "'hello'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSQLValue(tt.colType, tt.value)
			if got != tt.want {
				t.Errorf("formatSQLValue(%q, %q) = %q, want %q", tt.colType, tt.value, got, tt.want)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple",
			in:   "users",
			want: `"users"`,
		},
		{
			name: "reserved",
			in:   "order",
			want: `"order"`,
		},
		{
			name: "with_space",
			in:   "my column",
			want: `"my column"`,
		},
		{
			name: "double_quote_escaped",
			in:   `foo"bar`,
			want: `"foo""bar"`,
		},
		{
			name: "empty",
			in:   "",
			want: `""`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteIdentifier(tt.in)
			if got != tt.want {
				t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildUpdateQueryFromCell(t *testing.T) {
	tests := []struct {
		name         string
		schema       string
		table        string
		setColumn    string
		setColType   string
		setValue     string
		pkColumns    []string
		row          map[string]string
		colTypeByKey map[string]string
		want         []string
		notWant      []string
	}{
		{
			name:         "basic_update_fallback_when_no_pk",
			table:        "users",
			setColumn:    "name",
			setColType:   "text",
			setValue:     "alice",
			row:          map[string]string{"name": "bob"},
			colTypeByKey: map[string]string{"name": "text"},
			want: []string{
				"WARNING",
				"WHERE uses the edited column",
				`UPDATE "users"`,
				`SET "name" = 'alice'`,
				`WHERE "name" = 'bob';`,
			},
			notWant: []string{`WHERE "name" = 'alice'`},
		},
		{
			name:         "integer_column_unquoted_literal",
			table:        "items",
			setColumn:    "id",
			setColType:   "integer",
			setValue:     "2",
			row:          map[string]string{"id": "1"},
			colTypeByKey: map[string]string{"id": "integer"},
			want: []string{
				"WARNING",
				`UPDATE "items"`,
				`SET "id" = 2`,
				`WHERE "id" = 1;`,
			},
			notWant: []string{`WHERE "id" = 2`},
		},
		{
			name:         "empty_value_produces_NULL",
			table:        "t",
			setColumn:    "opt",
			setColType:   "text",
			setValue:     "",
			row:          map[string]string{"opt": "old"},
			colTypeByKey: map[string]string{"opt": "text"},
			want: []string{
				"WARNING",
				`SET "opt" = NULL`,
				`WHERE "opt" = 'old';`,
			},
		},
		{
			name:         "real_column_unquoted",
			table:        "t",
			setColumn:    "price",
			setColType:   "real",
			setValue:     "9.99",
			row:          map[string]string{"price": "4.99"},
			colTypeByKey: map[string]string{"price": "real"},
			want: []string{
				"WARNING",
				`SET "price" = 9.99`,
				`WHERE "price" = 4.99;`,
			},
			notWant: []string{`WHERE "price" = 9.99`},
		},
		{
			name:         "text_with_quote_escaped",
			table:        "t",
			setColumn:    "name",
			setColType:   "text",
			setValue:     "O'Brien",
			row:          map[string]string{"name": "Smith"},
			colTypeByKey: map[string]string{"name": "text"},
			want: []string{
				"WARNING",
				`SET "name" = 'O''Brien'`,
				`WHERE "name" = 'Smith';`,
			},
		},
		{
			name:         "where_uses_primary_key_when_provided",
			table:        "users",
			setColumn:    "name",
			setColType:   "text",
			setValue:     "bob",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "1", "name": "alice"},
			colTypeByKey: map[string]string{"id": "integer", "name": "text"},
			want: []string{
				`UPDATE "users"`,
				`SET "name" = 'bob'`,
				`WHERE "id" = 1`,
			},
			notWant: []string{`WHERE "name"`},
		},
		{
			name:         "reserved_keyword_table_and_column_quoted",
			table:        "order",
			setColumn:    "select",
			setColType:   "text",
			setValue:     "x",
			row:          map[string]string{"select": "y"},
			colTypeByKey: map[string]string{"select": "text"},
			want: []string{
				"WARNING",
				`UPDATE "order"`,
				`SET "select" = 'x'`,
				`WHERE "select" = 'y';`,
			},
		},
		{
			name:         "identifier_with_double_quote_escaped",
			table:        "t",
			setColumn:    `foo"bar`,
			setColType:   "text",
			setValue:     "v",
			row:          map[string]string{`foo"bar`: "old"},
			colTypeByKey: map[string]string{`foo"bar`: "text"},
			want: []string{
				"WARNING",
				`"foo""bar"`,
				`SET "foo""bar" = 'v'`,
				`WHERE "foo""bar" = 'old';`,
			},
		},
		{
			name:         "identifier_with_space_quoted",
			table:        "my table",
			setColumn:    "my column",
			setColType:   "integer",
			setValue:     "2",
			row:          map[string]string{"my column": "1"},
			colTypeByKey: map[string]string{"my column": "integer"},
			want: []string{
				"WARNING",
				`UPDATE "my table"`,
				`SET "my column" = 2`,
				`WHERE "my column" = 1;`,
			},
			notWant: []string{`WHERE "my column" = 2`},
		},
		{
			name:         "malicious_identifier_quoted_not_injection",
			table:        "t",
			setColumn:    `"; DROP TABLE t; --`,
			setColType:   "text",
			setValue:     "x",
			row:          map[string]string{`"; DROP TABLE t; --`: "old"},
			colTypeByKey: map[string]string{`"; DROP TABLE t; --`: "text"},
			want: []string{
				"WARNING",
				`SET "`,
				`= 'x'`,
			},
		},
		{
			name:         "postgres_schema_qualifies_table",
			schema:       "public",
			table:        "users",
			setColumn:    "name",
			setColType:   "text",
			setValue:     "bob",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "1", "name": "alice"},
			colTypeByKey: map[string]string{"id": "integer", "name": "text"},
			want: []string{
				`UPDATE "public"."users"`,
				`SET "name" = 'bob'`,
				`WHERE "id" = 1`,
			},
			notWant: []string{`UPDATE "users"`},
		},
		{
			name:         "non_public_schema_qualifies_table",
			schema:       "analytics",
			table:        "events",
			setColumn:    "status",
			setColType:   "text",
			setValue:     "done",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "7", "status": "pending"},
			colTypeByKey: map[string]string{"id": "integer", "status": "text"},
			want: []string{
				`UPDATE "analytics"."events"`,
				`SET "status" = 'done'`,
				`WHERE "id" = 7`,
			},
			notWant: []string{`UPDATE "events"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildUpdateQueryFromCell(tt.schema, tt.table, tt.setColumn, tt.setColType, tt.setValue, tt.pkColumns, tt.row, tt.colTypeByKey)
			for _, sub := range tt.want {
				if !strings.Contains(got, sub) {
					t.Errorf("BuildUpdateQueryFromCell(...) result must contain %q.\nGot:\n%s", sub, got)
				}
			}
			for _, sub := range tt.notWant {
				if strings.Contains(got, sub) {
					t.Errorf("BuildUpdateQueryFromCell(...) result must not contain %q.\nGot:\n%s", sub, got)
				}
			}
		})
	}
}

func TestBuildDeleteQuery(t *testing.T) {
	tests := []struct {
		name         string
		schema       string
		table        string
		pkColumns    []string
		row          map[string]string
		colTypeByKey map[string]string
		want         []string
		notWant      []string
	}{
		{
			name:         "uses_pk_for_where_when_provided",
			table:        "users",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "1", "name": "alice"},
			colTypeByKey: map[string]string{"id": "integer", "name": "text"},
			want: []string{
				`DELETE FROM "users"`,
				`WHERE "id" = 1;`,
			},
			notWant: []string{"WARNING", `"name"`},
		},
		{
			name:         "composite_pk_uses_all_pk_columns",
			table:        "habit_logs",
			pkColumns:    []string{"user_id", "habit_id"},
			row:          map[string]string{"user_id": "1", "habit_id": "2", "note": "done"},
			colTypeByKey: map[string]string{"user_id": "integer", "habit_id": "integer", "note": "text"},
			want: []string{
				`DELETE FROM "habit_logs"`,
				`"user_id" = 1`,
				`"habit_id" = 2`,
			},
			notWant: []string{"WARNING", `"note"`},
		},
		{
			name:         "falls_back_to_all_columns_when_no_pk",
			table:        "logs",
			row:          map[string]string{"level": "info", "msg": "started"},
			colTypeByKey: map[string]string{"level": "text", "msg": "text"},
			want: []string{
				"WARNING",
				`DELETE FROM "logs"`,
				`"level" = 'info'`,
				`"msg" = 'started'`,
			},
		},
		{
			name:         "integer_pk_value_unquoted",
			table:        "items",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "42"},
			colTypeByKey: map[string]string{"id": "integer"},
			want: []string{
				`DELETE FROM "items"`,
				`WHERE "id" = 42;`,
			},
			notWant: []string{`"id" = '42'`},
		},
		{
			name:         "text_pk_value_with_quote_escaped",
			table:        "t",
			pkColumns:    []string{"code"},
			row:          map[string]string{"code": "O'Brien"},
			colTypeByKey: map[string]string{"code": "text"},
			want: []string{
				`DELETE FROM "t"`,
				`WHERE "code" = 'O''Brien';`,
			},
		},
		{
			name:         "reserved_keyword_table_quoted",
			table:        "order",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "1"},
			colTypeByKey: map[string]string{"id": "integer"},
			want:         []string{`DELETE FROM "order"`, `WHERE "id" = 1;`},
		},
		{
			name:         "malicious_identifier_quoted_not_injection",
			table:        "t",
			pkColumns:    []string{`"; DROP TABLE t; --`},
			row:          map[string]string{`"; DROP TABLE t; --`: "1"},
			colTypeByKey: map[string]string{`"; DROP TABLE t; --`: "integer"},
			want:         []string{`DELETE FROM "t"`, `WHERE "`},
		},
		{
			name:         "postgres_schema_qualifies_table",
			schema:       "public",
			table:        "users",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "5", "name": "alice"},
			colTypeByKey: map[string]string{"id": "integer", "name": "text"},
			want:         []string{`DELETE FROM "public"."users"`, `WHERE "id" = 5;`},
			notWant:      []string{`DELETE FROM "users"`},
		},
		{
			name:         "non_public_schema_qualifies_table",
			schema:       "myapp",
			table:        "orders",
			pkColumns:    []string{"id"},
			row:          map[string]string{"id": "99"},
			colTypeByKey: map[string]string{"id": "integer"},
			want:         []string{`DELETE FROM "myapp"."orders"`, `WHERE "id" = 99;`},
			notWant:      []string{`DELETE FROM "orders"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDeleteQuery(tt.schema, tt.table, tt.pkColumns, tt.row, tt.colTypeByKey)
			for _, sub := range tt.want {
				if !strings.Contains(got, sub) {
					t.Errorf("BuildDeleteQuery(...) result must contain %q.\nGot:\n%s", sub, got)
				}
			}
			for _, sub := range tt.notWant {
				if strings.Contains(got, sub) {
					t.Errorf("BuildDeleteQuery(...) result must not contain %q.\nGot:\n%s", sub, got)
				}
			}
		})
	}
}
