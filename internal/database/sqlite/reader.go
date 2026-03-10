package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jxdones/stoat/internal/database"
)

// paginationMode is the strategy for paginating a table (rowid, integer PK, or offset).
type paginationMode int

const (
	paginationByRowID paginationMode = iota
	paginationByIntegerPK
	paginationByOffset
)

// offsetCursorSkipCurrentRow is added to the offset when building the next-page cursor
// so the next page starts after the current row.
const offsetCursorSkipCurrentRow = 1

// Constants for cursor parsing and conversion.
const (
	decimalBase  = 10
	int64BitSize = 64
)

// Constants for column display.
const (
	minColumnWidth = 8
	maxColumnWidth = 24
	columnNamePad  = 2
)

// tableInfo holds column metadata for one table column.
type tableInfo struct {
	Name         string
	DeclaredType string
	NotNull      bool
	DefaultValue string
	PKOrder      int
}

// indexListRow holds one row from PRAGMA index_list.
type indexListRow struct {
	Name   string
	Unique bool
	Origin string
}

// rowsPagePlan holds the SQL and metadata for one page of table rows.
type rowsPagePlan struct {
	mode       paginationMode
	query      string
	scanOffset int
	afterValue int64
}

// Databases returns the list of database names in the given path.
func Databases(ctx context.Context, dbName string, path string) ([]string, error) {
	name := strings.TrimSpace(dbName)
	if name == "" {
		name = filepath.Base(path)
	}
	return []string{name}, nil
}

// Tables returns the list of table names in the given database.
func Tables(ctx context.Context, db *sql.DB) ([]string, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		value := strings.TrimSpace(name.String)
		if value != "" {
			tables = append(tables, value)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

// Rows returns a page of rows for the given table.
// The result includes columns, rows, and pagination state (HasMore, NextAfter).
func Rows(ctx context.Context, db *sql.DB, target database.DatabaseTarget, page database.PageRequest) (database.PageResult, error) {
	columnsInfo, err := tableInfoRows(ctx, db, target.Table)
	if err != nil {
		return database.PageResult{}, err
	}

	if len(columnsInfo) == 0 {
		return database.PageResult{}, errors.New("table has no columns")
	}

	pageLimit := page.Limit
	if pageLimit <= 0 {
		pageLimit = 200
	}

	columnNames, selectColumns := buildPageColumns(columnsInfo)
	plan, err := buildRowsPagePlan(ctx, db, target.Table, columnsInfo, columnNames, selectColumns, page, pageLimit)
	if err != nil {
		return database.PageResult{}, err
	}
	rows, hasMore, nextAfter, err := scanRowsPageResult(ctx, db, plan, columnNames, pageLimit)
	if err != nil {
		return database.PageResult{}, err
	}
	return database.PageResult{
		Result: database.QueryResult{
			Columns: buildOutputColumns(columnsInfo),
			Rows:    rows,
		},
		StartAfter: plan.afterValue,
		HasMore:    hasMore,
		NextAfter:  nextAfter,
	}, nil
}

// buildPageColumns returns column names and quoted SELECT column list for the given table columns.
func buildPageColumns(columnsInfo []tableInfo) ([]string, []string) {
	columnNames := make([]string, 0, len(columnsInfo))
	selectColumns := make([]string, 0, len(columnsInfo))
	for _, column := range columnsInfo {
		columnNames = append(columnNames, column.Name)
		selectColumns = append(selectColumns, quoteIdentifier(column.Name))
	}
	return columnNames, selectColumns
}

// buildRowsPagePlan builds the SQL and metadata for one page of table rows.
// It chooses a strategy based on the table: rowid, single integer primary key, or offset.
// Returns an error if the table cannot be inspected or the cursor is invalid.
func buildRowsPagePlan(ctx context.Context, db *sql.DB, table string, columnsInfo []tableInfo, columnNames, selectColumns []string, page database.PageRequest, pageLimit int) (rowsPagePlan, error) {
	cursorAlias := uniqueCursorAlias(columnNames)
	primaryKeyColumns := orderedPrimaryKeyColumns(columnsInfo)
	withoutRowID, err := isTableWithoutRowID(ctx, db, table)
	if err != nil {
		return rowsPagePlan{}, err
	}

	// Request one extra row to know if there is a next page (HasMore).
	limit := pageLimit + 1
	switch {
	case !withoutRowID:
		after, err := parseCursor(page.After, "rowid")
		if err != nil {
			return rowsPagePlan{}, err
		}
		where := ""
		if after > 0 {
			where = fmt.Sprintf("WHERE rowid > %d", after)
		}
		return rowsPagePlan{
			mode: paginationByRowID,
			query: fmt.Sprintf(
				"SELECT rowid AS %s, %s FROM %s %s ORDER BY rowid LIMIT %d;",
				quoteIdentifier(cursorAlias),
				strings.Join(selectColumns, ", "),
				quoteIdentifier(table),
				where,
				limit,
			),
			scanOffset: 1, // first column is the rowid cursor
			afterValue: after,
		}, nil
	case len(primaryKeyColumns) == 1 && hasIntegerAffinity(columnDeclaredType(columnsInfo, primaryKeyColumns[0])):
		after, err := parseCursor(page.After, "pk")
		if err != nil {
			return rowsPagePlan{}, err
		}
		primaryKey := quoteIdentifier(primaryKeyColumns[0])
		where := ""
		if after > 0 {
			where = fmt.Sprintf("WHERE %s > %d", primaryKey, after)
		}
		return rowsPagePlan{
			mode: paginationByIntegerPK,
			query: fmt.Sprintf(
				"SELECT %s AS %s, %s FROM %s %s ORDER BY %s LIMIT %d;",
				primaryKey,
				quoteIdentifier(cursorAlias),
				strings.Join(selectColumns, ", "),
				quoteIdentifier(table),
				where,
				primaryKey,
				limit,
			),
			scanOffset: 1, // first column is the PK cursor
			afterValue: after,
		}, nil
	default:
		offset, err := parseCursor(page.After, "off")
		if err != nil {
			return rowsPagePlan{}, err
		}
		if offset < 0 {
			offset = 0
		}
		return rowsPagePlan{
			mode: paginationByOffset,
			query: fmt.Sprintf(
				"SELECT %s FROM %s ORDER BY %s LIMIT %d OFFSET %d;",
				strings.Join(selectColumns, ", "),
				quoteIdentifier(table),
				primaryKeyOrderExpr(primaryKeyColumns),
				limit,
				offset,
			),
			scanOffset: 0, // no separate cursor column; offset is the cursor
			afterValue: offset,
		}, nil
	}
}

// scanRowsPageResult executes the plan's query and returns up to pageLimit rows.
// The boolean reports whether more rows exist; the string is an opaque cursor for the next page.
func scanRowsPageResult(ctx context.Context, db *sql.DB, plan rowsPagePlan, columnNames []string, pageLimit int) ([]database.Row, bool, string, error) {
	rows, err := db.QueryContext(ctx, plan.query)
	if err != nil {
		return nil, false, "", err
	}
	defer rows.Close()

	values, targets := makeScanBuffers(len(columnNames) + plan.scanOffset) // allocate an extra slot for the cursor column if needed
	outputRows := make([]database.Row, 0, pageLimit)
	hasMore := false
	nextAfter := ""

	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return nil, false, "", err
		}
		if len(outputRows) >= pageLimit {
			hasMore = true
			break
		}
		switch plan.mode {
		case paginationByOffset:
			nextAfter = formatCursor("off", plan.afterValue+int64(len(outputRows))+offsetCursorSkipCurrentRow)
		case paginationByIntegerPK:
			nextAfter = formatCursor("pk", asInt64(values[0]))
		default:
			nextAfter = formatCursor("rowid", asInt64(values[0]))
		}
		row := make(database.Row, len(columnNames))
		for i, name := range columnNames {
			row[name] = asString(values[i+plan.scanOffset])
		}
		outputRows = append(outputRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, false, "", err
	}
	return outputRows, hasMore, nextAfter, nil
}

// Query executes a single SQL statement and returns a normalized result.
// For SELECT it returns columns and rows (capped at 1000); each row is a map from column name to string.
// For INSERT/UPDATE/DELETE it sets RowsAffected from SQLite's changes().
// Returns query result or an error if the query is invalid or the connection is lost.
func Query(ctx context.Context, db *sql.DB, query string) (database.QueryResult, error) {
	const maxRows = 1000
	const queryResultCap = 256

	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return database.QueryResult{}, err
	}
	defer sqlConn.Close()

	rows, err := sqlConn.QueryContext(ctx, query)
	if err != nil {
		return database.QueryResult{}, err
	}

	columnNames, err := rows.Columns()
	if err != nil {
		rows.Close()
		return database.QueryResult{}, err
	}

	resultRows := make([]database.Row, 0, min(maxRows, queryResultCap))
	values, targets := makeScanBuffers(len(columnNames))

	for rows.Next() {
		if len(resultRows) >= maxRows {
			break
		}
		if err := rows.Scan(targets...); err != nil {
			rows.Close()
			return database.QueryResult{}, err
		}
		row := make(database.Row, len(columnNames))
		for i, name := range columnNames {
			row[name] = asString(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return database.QueryResult{}, err
	}
	if err := rows.Close(); err != nil {
		return database.QueryResult{}, err
	}

	seen := make(map[string]struct{}, len(columnNames))
	orderedColumns := make([]string, 0, len(columnNames))
	for _, name := range columnNames {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		orderedColumns = append(orderedColumns, name)
	}

	columns := make([]database.Column, 0, len(orderedColumns))
	for i, column := range orderedColumns {
		columns = append(columns, database.Column{
			Key:      column,
			Title:    column,
			Type:     "text", // we don't infer types from ad-hoc queries; "text" is safe for display
			MinWidth: max(minColumnWidth, min(maxColumnWidth, len([]rune(column))+columnNamePad)),
			Order:    i + 1, // 1-based position so UI can sort columns (e.g. Order 1 = first column)
		})
	}

	var affectedRows int64
	_ = sqlConn.QueryRowContext(ctx, "SELECT changes();").Scan(&affectedRows)

	return database.QueryResult{
		Columns:      columns,
		Rows:         resultRows,
		RowsAffected: affectedRows,
	}, nil
}

// Indexes returns the list of indexes for the given table.
func Indexes(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.Index, error) {
	indexRows, err := indexListRows(ctx, db, target.Table)
	if err != nil {
		return nil, err
	}

	result := make([]database.Index, 0, len(indexRows))
	for _, row := range indexRows {
		if strings.TrimSpace(row.Name) == "" {
			continue
		}
		columns, err := indexColumns(ctx, db, row.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, database.Index{
			Name:    row.Name,
			Columns: columns,
			Unique:  row.Unique,
		})
	}
	return result, nil
}

// Constraints returns the list of constraints for the given table.
func Constraints(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.Constraint, error) {
	infoRows, err := tableInfoRows(ctx, db, target.Table)
	if err != nil {
		return nil, err
	}
	result := make([]database.Constraint, 0, len(infoRows)+2)

	primaryKeyByOrder := map[int]string{}
	for _, row := range infoRows {
		column := strings.TrimSpace(row.Name)
		if column == "" {
			continue
		}
		if row.NotNull {
			result = append(result, database.Constraint{
				Name:    "NOT NULL " + column,
				Type:    "NOT NULL",
				Columns: []string{column},
			})
		}
		if defaultValue := strings.TrimSpace(row.DefaultValue); defaultValue != "" {
			result = append(result, database.Constraint{
				Name:    "DEFAULT " + column,
				Type:    "DEFAULT",
				Columns: []string{column},
				Detail:  defaultValue,
			})
		}
		if row.PKOrder > 0 {
			primaryKeyByOrder[row.PKOrder] = column
		}
	}

	if len(primaryKeyByOrder) > 0 {
		order := make([]int, 0, len(primaryKeyByOrder))
		for key := range primaryKeyByOrder {
			order = append(order, key)
		}
		sort.Ints(order)
		columns := make([]string, 0, len(order))
		for _, key := range order {
			columns = append(columns, primaryKeyByOrder[key])
		}
		// Primary key constraint should be always first.
		result = append([]database.Constraint{{
			Name:    "PRIMARY KEY",
			Type:    "PRIMARY KEY",
			Columns: columns,
		}}, result...)
	}

	indexRows, err := indexListRows(ctx, db, target.Table)
	if err != nil {
		return nil, err
	}
	for _, row := range indexRows {
		if !row.Unique {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row.Origin), "pk") {
			continue
		}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		columns, err := indexColumns(ctx, db, name)
		if err != nil {
			return nil, err
		}
		result = append(result, database.Constraint{
			Name:    name,
			Type:    "UNIQUE",
			Columns: columns,
		})
	}

	return result, nil
}

// ForeignKeys returns the list of foreign keys for the given table.
func ForeignKeys(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.ForeignKey, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s);", quoteIdentifier(target.Table))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]database.ForeignKey, 0)
	for rows.Next() {
		var (
			id         int64
			seq        int64
			refTable   sql.NullString
			fromColumn sql.NullString
			toColumn   sql.NullString
			onUpdate   sql.NullString
			onDelete   sql.NullString
			match      sql.NullString
		)
		if err := rows.Scan(&id, &seq, &refTable, &fromColumn, &toColumn, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		column := strings.TrimSpace(fromColumn.String)
		refTableValue := strings.TrimSpace(refTable.String)
		refColumn := strings.TrimSpace(toColumn.String)
		if column == "" || refTableValue == "" || refColumn == "" {
			continue
		}
		result = append(result, database.ForeignKey{
			Name:           fmt.Sprintf("fk_%s_%s", target.Table, column),
			Column:         column,
			RefTable:       refTableValue,
			RefColumn:      refColumn,
			OnUpdateAction: strings.TrimSpace(onUpdate.String),
			OnDeleteAction: strings.TrimSpace(onDelete.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// tableInfoRows returns the column metadata for the given table.
func tableInfoRows(ctx context.Context, db *sql.DB, table string) ([]tableInfo, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s);", quoteIdentifier(table))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]tableInfo, 0)
	for rows.Next() {
		var (
			columnID     int64
			name         sql.NullString
			declaredType sql.NullString
			notNull      int64
			defaultValue sql.NullString
			primaryKey   int64
		)
		if err := rows.Scan(&columnID, &name, &declaredType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		nameValue := strings.TrimSpace(name.String)
		if nameValue == "" {
			continue
		}
		result = append(result, tableInfo{
			Name:         nameValue,
			DeclaredType: strings.TrimSpace(declaredType.String),
			NotNull:      notNull == 1,
			DefaultValue: strings.TrimSpace(defaultValue.String),
			PKOrder:      int(primaryKey),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// indexListRows returns the list of indexes for the given table.
func indexListRows(ctx context.Context, db *sql.DB, table string) ([]indexListRow, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s);", quoteIdentifier(table))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]indexListRow, 0)
	for rows.Next() {
		var (
			seq     int64
			name    sql.NullString
			unique  int64
			origin  sql.NullString
			partial int64
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}
		result = append(result, indexListRow{
			Name:   strings.TrimSpace(name.String),
			Unique: unique == 1,
			Origin: strings.TrimSpace(origin.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// indexColumns returns the column names of the given index in order.
func indexColumns(ctx context.Context, db *sql.DB, indexName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA index_info(%s);", quoteIdentifier(indexName))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ordered := make([]struct {
		sequence int
		column   string
	}, 0)

	for rows.Next() {
		var (
			seqno int64
			cid   int64
			name  sql.NullString
		)
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, err
		}
		column := strings.TrimSpace(name.String)
		if column == "" {
			continue
		}
		ordered = append(ordered, struct {
			sequence int
			column   string
		}{
			sequence: int(seqno),
			column:   column,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].sequence < ordered[j].sequence
	})

	result := make([]string, 0, len(ordered))
	for _, item := range ordered {
		result = append(result, item.column)
	}
	return result, nil
}

// quoteIdentifier quotes an identifier for use in an SQL query.
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// quoteSQLString quotes a string for use in an SQL query.
func quoteSQLString(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

// orderedPrimaryKeyColumns returns the primary key column names in table order.
// Non-PK columns and columns with empty names are ignored.
func orderedPrimaryKeyColumns(columnsInfo []tableInfo) []string {
	orderToColumn := make(map[int]string, len(columnsInfo))
	for _, column := range columnsInfo {
		if column.PKOrder <= 0 {
			continue
		}
		name := strings.TrimSpace(column.Name)
		if name == "" {
			continue
		}
		orderToColumn[column.PKOrder] = name
	}
	ordered := make([]int, 0, len(orderToColumn))
	for order := range orderToColumn {
		ordered = append(ordered, order)
	}
	sort.Ints(ordered)
	orderedColumns := make([]string, 0, len(ordered))

	for _, order := range ordered {
		orderedColumns = append(orderedColumns, orderToColumn[order])
	}

	return orderedColumns
}

// primaryKeyOrderExpr returns an ORDER BY expression for the primary key columns, or "1" if none.
func primaryKeyOrderExpr(primaryKeyColumns []string) string {
	if len(primaryKeyColumns) == 0 {
		return "1"
	}
	out := make([]string, 0, len(primaryKeyColumns))
	for _, column := range primaryKeyColumns {
		out = append(out, quoteIdentifier(column))
	}
	return strings.Join(out, ", ")
}

// uniqueCursorAlias returns an alias for the cursor column that does not conflict with existing column names.
func uniqueCursorAlias(columns []string) string {
	used := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		used[strings.ToLower(strings.TrimSpace(column))] = struct{}{}
	}
	alias := "__cursor"
	for {
		if _, ok := used[strings.ToLower(alias)]; !ok {
			return alias
		}
		alias += "_"
	}
}

// parseCursor parses the cursor string and returns the integer value for the given prefix.
// An empty cursor returns 0, nil.
func parseCursor(cursor string, expectedPrefix string) (int64, error) {
	raw := strings.TrimSpace(cursor)
	if raw == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(raw, decimalBase, int64BitSize); err == nil {
		return n, nil
	}

	prefix := expectedPrefix + ":"
	if !strings.HasPrefix(raw, prefix) {
		return 0, fmt.Errorf("invalid cursor %q for mode %s", raw, expectedPrefix)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(raw, prefix)), decimalBase, int64BitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor %q for mode %s: %w", raw, expectedPrefix, err)
	}
	return n, nil
}

// formatCursor formats a cursor as "prefix:n".
func formatCursor(prefix string, n int64) string {
	return fmt.Sprintf("%s:%d", prefix, n)
}

// isTableWithoutRowID reports whether the table was created with WITHOUT ROWID.
func isTableWithoutRowID(ctx context.Context, dbConn *sql.DB, table string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT sql FROM sqlite_master WHERE type='table' AND name=%s LIMIT 1;",
		quoteSQLString(strings.TrimSpace(table)),
	)
	var createStatement sql.NullString
	if err := dbConn.QueryRowContext(ctx, query).Scan(&createStatement); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	stmt := strings.ToUpper(strings.TrimSpace(createStatement.String))
	return strings.Contains(stmt, "WITHOUT ROWID"), nil
}

// hasIntegerAffinity reports whether the declared type has integer affinity (e.g. INT, INTEGER).
func hasIntegerAffinity(declaredType string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(declaredType)), "INT")
}

// columnDeclaredType returns the declared type of the given column, or empty string if not found.
func columnDeclaredType(columnsInfo []tableInfo, column string) string {
	for _, info := range columnsInfo {
		if !strings.EqualFold(strings.TrimSpace(info.Name), column) {
			continue
		}
		return strings.TrimSpace(info.DeclaredType)
	}
	return ""
}

// makeScanBuffers allocates n value slots and n pointers (targets[i] == &values[i]) for rows.Scan.
// After calling rows.Scan(targets...), the row data lives in values.
func makeScanBuffers(n int) ([]any, []any) {
	values := make([]any, n)
	targets := make([]any, n)
	for i := range values {
		targets[i] = &values[i]
	}
	return values, targets
}

// buildOutputColumns builds the display column list for the given table columns.
func buildOutputColumns(columnsInfo []tableInfo) []database.Column {
	columns := make([]database.Column, 0, len(columnsInfo))
	for i, column := range columnsInfo {
		declaredType := strings.TrimSpace(column.DeclaredType)
		if declaredType == "" {
			declaredType = "text"
		}
		widthFromName := len([]rune(column.Name)) + columnNamePad
		columns = append(columns, database.Column{
			Key:      column.Name,
			Title:    column.Name,
			Type:     strings.ToLower(declaredType),
			MinWidth: max(minColumnWidth, min(maxColumnWidth, widthFromName)),
			Order:    i + 1, // preserve table column order (1-based for display)
		})
	}
	return columns
}

// asString converts a scanned value to a string for display.
func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case float64:
		if float64(int64(t)) == t {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", t)
	}
}

// asInt64 converts a scanned value to int64 for cursor use.
func asInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(strings.TrimSpace(string(t)), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}
