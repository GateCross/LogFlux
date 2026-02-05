package log

import "fmt"

var errInvalidLogSourcePath = fmt.Errorf("log source path is required")
var errInvalidLogSourceScanInterval = fmt.Errorf("scan interval must be > 0 seconds (default 60)")
