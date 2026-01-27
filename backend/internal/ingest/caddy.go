package ingest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"logflux/model"

	"github.com/nxadm/tail"

	"gorm.io/gorm"
)

// Log Format:
// [{ts}] "{country_name}" "{province_name}" "{city_name}" "{request>host}" "{request>method} {request>uri} {request>proto}" {status} {size} "{request>headers>User-Agent>[0]}" "{request>remote_ip}" "{request>client_ip}"

var logRegex = regexp.MustCompile(`^\[(.*?)\] "(.*?)" "(.*?)" "(.*?)" "(.*?)" "(.*?) (.*?) (.*?)" (\d+) (\d+) "(.*?)" "(.*?)" "(.*?)"$`)

type CaddyIngestor struct {
	db    *gorm.DB
	tails map[string]*tail.Tail
	mu    sync.Mutex
}

func NewCaddyIngestor(db *gorm.DB) *CaddyIngestor {
	return &CaddyIngestor{
		db:    db,
		tails: make(map[string]*tail.Tail),
	}
}

func (i *CaddyIngestor) ParseLine(line string) (*model.CaddyLog, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	matches := logRegex.FindStringSubmatch(line)
	if len(matches) != 14 {
		return nil, fmt.Errorf("invalid log format: %s", line)
	}

	logTime, err := i.parseTime(matches[1])
	if err != nil {
		fmt.Printf("Time parse error: %v for %s\n", err, matches[1])
	}

	status, _ := strconv.Atoi(matches[9])
	size, _ := strconv.ParseInt(matches[10], 10, 64)

	return &model.CaddyLog{
		LogTime:   logTime,
		Country:   matches[2],
		Province:  matches[3],
		City:      matches[4],
		Host:      matches[5],
		Method:    matches[6],
		Uri:       matches[7],
		Proto:     matches[8],
		Status:    status,
		Size:      size,
		UserAgent: matches[11],
		RemoteIP:  matches[12],
		ClientIP:  matches[13],
	}, nil
}

func (i *CaddyIngestor) parseTime(ts string) (time.Time, error) {
	layouts := []string{
		"2006/01/02 15:04:05.000",
		"02/Jan/2006:15:04:05 -0700",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, ts); err == nil {
			return t, nil
		}
		if t, err := time.ParseInLocation(layout, ts, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Now(), fmt.Errorf("unknown time format")
}

func (i *CaddyIngestor) Ingest(line string) error {
	logEntry, err := i.ParseLine(line)
	if err != nil {
		return err
	}
	return i.db.Create(logEntry).Error
}

func (i *CaddyIngestor) Start(filePath string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.tails[filePath]; exists {
		// Already runing
		return
	}

	go func() {
		t, err := tail.TailFile(filePath, tail.Config{
			Follow: true,
			ReOpen: true,
			Poll:   true,
		})
		if err != nil {
			fmt.Printf("Error tailing file: %v\n", err)
			return
		}

		i.mu.Lock()
		i.tails[filePath] = t
		i.mu.Unlock()

		fmt.Printf("Started monitoring: %s\n", filePath)

		for line := range t.Lines {
			if err := i.Ingest(line.Text); err != nil {
				// Verbose error logging might flood
			}
		}
	}()
}

func (i *CaddyIngestor) Stop(filePath string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if t, exists := i.tails[filePath]; exists {
		t.Stop()
		t.Cleanup()
		delete(i.tails, filePath)
		fmt.Printf("Stopped monitoring: %s\n", filePath)
	}
}
