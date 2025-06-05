package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StringMap map[string]string

func (m StringMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]string)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal StringMap value: %v", value)
	}
	return json.Unmarshal(bytes, m)
}
