package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
)

func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(stringSliceConverter{}))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	mock.MatchExpectationsInOrder(false)
	mock.ExpectClose()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	return &DB{log: slog.Default(), conn: sqlxDB}, mock, func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	}
}

type stringSliceConverter struct{}

func (stringSliceConverter) ConvertValue(v interface{}) (driver.Value, error) {
	if val, ok := v.([]string); ok {
		return strings.Join(val, ","), nil
	}
	return driver.DefaultParameterConverter.ConvertValue(v)
}

func TestAdd(t *testing.T) {
	storage, mock, cleanup := newMockDB(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO comics (id, url, words) VALUES($1, $2, $3)")).
		WithArgs(1, "url", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.Add(context.Background(), core.Comics{ID: 1, URL: "url", Words: []string{"a", "b"}}); err != nil {
		t.Fatalf("add failed: %v", err)
	}
}

func TestStats(t *testing.T) {
	storage, mock, cleanup := newMockDB(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM comics")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT coalesce(SUM(array_length(words,1)), 0) FROM comics")).WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(5))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM (SELECT distinct(unnest(words)) FROM comics)")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	stats, err := storage.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.ComicsFetched != 3 || stats.WordsTotal != 5 || stats.WordsUnique != 2 {
		t.Fatalf("unexpected stats %+v", stats)
	}
}

func TestIDs(t *testing.T) {
	storage, mock, cleanup := newMockDB(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM comics")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))

	ids, err := storage.IDs(context.Background())
	if err != nil {
		t.Fatalf("ids failed: %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("unexpected ids %v", ids)
	}
}

func TestDrop(t *testing.T) {
	storage, mock, cleanup := newMockDB(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("TRUNCATE comics")).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := storage.Drop(context.Background()); err != nil {
		t.Fatalf("drop failed: %v", err)
	}
}

func TestMigrate_BadConnection(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			t.Fatalf("close db: %v", cerr)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	}()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectClose()
	mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	storage := &DB{
		log:  slog.Default(),
		conn: sqlx.NewDb(db, "pgx"),
	}

	if err := storage.Migrate(); err == nil {
		t.Fatalf("expected migration to fail on bad connection")
	}
}

func TestClose(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	}()
	mock.ExpectClose()

	storage := &DB{
		log:  slog.Default(),
		conn: sqlx.NewDb(db, "sqlmock"),
	}

	if err := storage.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNew_Failure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := New(logger, "postgres://127.0.0.1:0/fail"); err == nil {
		t.Fatalf("expected connection error")
	}
}
