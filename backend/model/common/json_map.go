package common

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONMap 用于 GORM JSONB 字段。
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("解析 JSONB 值失败")
	}

	return json.Unmarshal(bytes, j)
}
