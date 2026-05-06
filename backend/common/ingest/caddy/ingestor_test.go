package caddy

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	caddymodel "logflux/model/caddy"
	ingestmodel "logflux/model/ingest"
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

func TestIngestWithPath_InternalAccessWritesSystemLog(t *testing.T) {
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
	line := `{"ts":1770000000.1,"request":{"host":"192.168.50.10:80","method":"GET","uri":"/api/health","proto":"HTTP/2.0","remote_ip":"127.0.0.1","client_ip":"","headers":{"User-Agent":["curl/8.0"]}},"status":200,"size":2}`

	mock.ExpectQuery(`INSERT INTO "system_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	if err := ingestor.IngestWithPath("/var/log/caddy/access.log", line); err != nil {
		t.Fatalf("IngestWithPath() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestIngestWithPath_PublicAccessWritesCaddyLog(t *testing.T) {
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
	line := `{"ts":1770000000.1,"request":{"host":"example.com","method":"GET","uri":"/","proto":"HTTP/2.0","remote_ip":"172.18.0.2","client_ip":"8.8.8.8","headers":{"User-Agent":["Mozilla/5.0"]}},"status":200,"size":128}`

	mock.ExpectQuery(`INSERT INTO "caddy_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	if err := ingestor.IngestWithPath("/var/log/caddy/access.log", line); err != nil {
		t.Fatalf("IngestWithPath() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestIsInternalCaddyAccess(t *testing.T) {
	tests := []struct {
		name  string
		entry caddymodel.CaddyLog
		want  bool
	}{
		{
			name:  "private host",
			entry: caddymodel.CaddyLog{Host: "192.168.50.10:80", RemoteIP: "8.8.8.8"},
			want:  true,
		},
		{
			name:  "localhost host",
			entry: caddymodel.CaddyLog{Host: "localhost", RemoteIP: "8.8.8.8"},
			want:  true,
		},
		{
			name:  "loopback ipv6 host",
			entry: caddymodel.CaddyLog{Host: "[::1]:80", RemoteIP: "8.8.8.8"},
			want:  true,
		},
		{
			name:  "private client ip",
			entry: caddymodel.CaddyLog{Host: "example.com", RemoteIP: "172.18.0.2", ClientIP: "10.0.0.2"},
			want:  true,
		},
		{
			name:  "private remote ip without client ip",
			entry: caddymodel.CaddyLog{Host: "example.com", RemoteIP: "192.168.50.10"},
			want:  false,
		},
		{
			name:  "public client behind private proxy",
			entry: caddymodel.CaddyLog{Host: "example.com", RemoteIP: "172.18.0.2", ClientIP: "8.8.8.8"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInternalCaddyAccess(&tt.entry); got != tt.want {
				t.Fatalf("isInternalCaddyAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

var _ = ingestmodel.LogIngestCursor{}
