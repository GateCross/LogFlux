package template

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MockEvent struct {
	Level     string
	Title     string
	Message   string
	Timestamp time.Time
	Data      map[string]interface{}
}

func TestTemplateManager_Render(t *testing.T) {
	// Setup mock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm connection: %v", err)
	}

	tm := NewTemplateManager(gdb)

	// Mock DB behavior for LoadTemplates
	rows := sqlmock.NewRows([]string{"name", "content", "format"}).
		AddRow("test_tmpl", "Hello {{.Title}}, {{.Message}}", "text")
	mock.ExpectQuery("^SELECT (.+) FROM \"notification_templates\"").WillReturnRows(rows)

	// Test LoadTemplates
	if err := tm.LoadTemplates(); err != nil {
		t.Errorf("LoadTemplates() error = %v", err)
	}

	// Test data
	event := MockEvent{
		Level:     "info",
		Title:     "World",
		Message:   "This is a test",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "val"},
	}

	// Test Render
	got, err := tm.Render("test_tmpl", event)
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	want := "Hello World, This is a test"
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}

	// Test Default Template
	gotDefault, err := tm.Render("default_text", event)
	if err != nil {
		t.Errorf("Render() default error = %v", err)
	}
	if len(gotDefault) == 0 {
		t.Error("Render() default returned empty string")
	}
}
