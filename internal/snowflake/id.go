// Package snowflake provides a Mastodon compatible Snowflate ID generator.
package snowflake

import (
	"time"
)

// TimeToID converts a time.Time to a Snowflake ID.
func TimeToID(ts time.Time) uint64 {
	// 48 bits for time in milliseconds.
	// 0 bits for worker ID.
	// 0 bits for sequence.
	// 16 bits for random. // TODO: use crypto/rand
	return uint64(ts.UnixNano()/int64(time.Millisecond))<<16 | uint64(time.Now().Nanosecond()&0xffff)
}

// IDToTime converts a Snowflake ID to a time.Time.
func IDToTime(id uint64) time.Time {
	return time.Unix(0, int64(id>>16)*1e6)
}
