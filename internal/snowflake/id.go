// Package snowflake provides a Mastodon compatible Snowflake ID generator.
package snowflake

import (
	"time"
)

// ID is a Mastodon compatible Snowflake ID.
// TODO use ID type instead of uint64 in the codebase.
type ID uint64

// TODO implement ID MarshalJSON and UnmarshalJSON methods.

// TimeToID converts a time.Time to a Snowflake ID.
func TimeToID(ts time.Time) ID {
	// 48 bits for time in milliseconds.
	// 0 bits for worker ID.
	// 0 bits for sequence.
	// 16 bits for random. // TODO: use crypto/rand
	return ID(uint64(ts.UnixNano()/int64(time.Millisecond))<<16 | uint64(time.Now().Nanosecond()&0xffff))
}

// IDToTime converts a Snowflake ID to a time.Time.
func (id ID) IDToTime() time.Time {
	return time.Unix(0, int64(id>>16)*1e6)
}

// Now returns the current time as a Snowflake ID.
func Now() ID {
	return TimeToID(time.Now())
}

// // GormDataType gorm common data type
// func (ID) GormDataType() string {
// 	return "uint64"
// }

// // GormDBDataType gorm db data type
// func (ID) GormDBDataType(db *gorm.DB, field *schema.Field) string {
// 	switch db.Dialector.Name() {
// 	case "mysql":
// 		return "BIGINT UNSIGNED"
// 	default:
// 		return ""
// 	}
// }

// func (id ID) Value() (driver.Value, error) {
// 	return uint64(id), nil
// }
