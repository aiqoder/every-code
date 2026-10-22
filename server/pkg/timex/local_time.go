package timex

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Layout 对外 JSON 时间格式。
const Layout = "2006-01-02 15:04:05"

// LocalTime 序列化为 Layout 的本地时间，兼容 GORM CreatedAt/UpdatedAt。
type LocalTime time.Time

func (t LocalTime) Time() time.Time {
	return time.Time(t)
}

func (t LocalTime) IsZero() bool {
	return time.Time(t).IsZero()
}

func (t LocalTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + tt.In(time.Local).Format(Layout) + `"`), nil
}

func (t *LocalTime) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == `""` {
		*t = LocalTime{}
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	parsed, err := time.ParseInLocation(Layout, s, time.Local)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return err
		}
	}
	*t = LocalTime(parsed)
	return nil
}

func (t LocalTime) Value() (driver.Value, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return nil, nil
	}
	return tt, nil
}

func (t *LocalTime) Scan(v interface{}) error {
	if v == nil {
		*t = LocalTime{}
		return nil
	}
	switch val := v.(type) {
	case time.Time:
		*t = LocalTime(val)
		return nil
	default:
		return fmt.Errorf("timex.LocalTime: cannot scan type %T", v)
	}
}

// GormDataType 返回 GORM 语义类型 time，由驱动映射为 datetime 列。
// 若直接返回 SQL 类型 datetime，GORM 不会把该字段当作时间，CreatedAt/UpdatedAt 也不会自动填充。
func (LocalTime) GormDataType() string {
	return "time"
}
