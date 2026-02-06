package ingest

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/model"
)

func TestResolveStartOffset_UsesCursorOffset(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	ingestor := NewCaddyIngestor(gdb)

	filePath := "/tmp/logflux-cursor-not-exist.log"
	mock.ExpectQuery(`SELECT \* FROM "log_ingest_cursors" WHERE file_path = \$1 LIMIT \$2`).
		WithArgs(filePath, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "file_path", "offset"}).AddRow(1, filePath, int64(128)))

	offset := ingestor.resolveStartOffset(filePath)
	if offset != 128 {
		t.Fatalf("expected offset=128, got %d", offset)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestResolveStartOffset_ReturnsZeroWhenCursorMissing(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	ingestor := NewCaddyIngestor(gdb)

	filePath := "/tmp/logflux-cursor-missing.log"
	mock.ExpectQuery(`SELECT \* FROM "log_ingest_cursors" WHERE file_path = \$1 LIMIT \$2`).
		WithArgs(filePath, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	offset := ingestor.resolveStartOffset(filePath)
	if offset != 0 {
		t.Fatalf("expected offset=0, got %d", offset)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestSaveOffset_UpsertCursor(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	ingestor := NewCaddyIngestor(gdb)

	filePath := "/tmp/logflux-cursor-save.log"
	mock.ExpectQuery(`INSERT INTO "log_ingest_cursors"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), filePath, int64(256), int64(256), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	if err := ingestor.saveOffset(filePath, 256); err != nil {
		t.Fatalf("saveOffset() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

var _ = model.LogIngestCursor{}
