package datasource

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jxdones/stoat/internal/database"
)

// timingDataSource wraps a DataSource and logs the duration of every method
// call to out. It is intended for debug builds only — enable via --debug flag.
type timingDataSource struct {
	source DataSource
	out    io.Writer
}

// WithTiming wraps source so that every method call is timed and written to out.
func WithTiming(source DataSource, out io.Writer) DataSource {
	return &timingDataSource{source: source, out: out}
}

// logTiming writes a single timing line to out. It is designed to be called
// with defer so that start is captured at call-site and elapsed is computed
// on return:
//
//	defer t.logTiming("MethodName", time.Now())
func (t *timingDataSource) logTiming(method string, start time.Time) {
	_, _ = fmt.Fprintf(t.out, "%s\t%s\t%s\n", time.Now().Format("2006-01-02 15:04:05"), method, time.Since(start))
}

// logTimingWithTarget writes a timing line to out with the target included.
func (t *timingDataSource) logTimingWithTarget(method, target string, start time.Time) {
	_, _ = fmt.Fprintf(t.out, "%s\t%s(%s)\t%s\n", time.Now().Format("2006-01-02 15:04:05"), method, target, time.Since(start))
}

// DefaultDatabase returns the default database name.
func (t *timingDataSource) DefaultDatabase(ctx context.Context) (string, error) {
	defer t.logTiming("DefaultDatabase", time.Now())
	return t.source.DefaultDatabase(ctx)
}

// Databases returns the list of databases.
func (t *timingDataSource) Databases(ctx context.Context) ([]string, error) {
	defer t.logTiming("Databases", time.Now())
	return t.source.Databases(ctx)
}

// Tables returns the list of tables in the given database.
func (t *timingDataSource) Tables(ctx context.Context, db string) ([]string, error) {
	defer t.logTiming("Tables", time.Now())
	return t.source.Tables(ctx, db)
}

// Rows returns a page of rows from the given table.
func (t *timingDataSource) Rows(ctx context.Context, target database.DatabaseTarget, page database.PageRequest) (database.PageResult, error) {
	targetName := fmt.Sprintf("%s.%s", target.Database, target.Table)
	defer t.logTimingWithTarget("Rows", targetName, time.Now())
	return t.source.Rows(ctx, target, page)
}

// Query executes a query and returns the result.
func (t *timingDataSource) Query(ctx context.Context, query string) (database.QueryResult, error) {
	defer t.logTiming("Query", time.Now())
	return t.source.Query(ctx, query)
}

// Indexes returns the list of indexes on the given table.
func (t *timingDataSource) Indexes(ctx context.Context, target database.DatabaseTarget) ([]database.Index, error) {
	targetName := fmt.Sprintf("%s.%s", target.Database, target.Table)
	defer t.logTimingWithTarget("Indexes", targetName, time.Now())
	return t.source.Indexes(ctx, target)
}

// Constraints returns the list of constraints on the given table.
func (t *timingDataSource) Constraints(ctx context.Context, target database.DatabaseTarget) ([]database.Constraint, error) {
	targetName := fmt.Sprintf("%s.%s", target.Database, target.Table)
	defer t.logTimingWithTarget("Constraints", targetName, time.Now())
	return t.source.Constraints(ctx, target)
}

// ForeignKeys returns the list of foreign keys on the given table.
func (t *timingDataSource) ForeignKeys(ctx context.Context, target database.DatabaseTarget) ([]database.ForeignKey, error) {
	targetName := fmt.Sprintf("%s.%s", target.Database, target.Table)
	defer t.logTimingWithTarget("ForeignKeys", targetName, time.Now())
	return t.source.ForeignKeys(ctx, target)
}

// Close closes the connection.
func (t *timingDataSource) Close() error {
	defer t.logTiming("Close", time.Now())
	return t.source.Close()
}
