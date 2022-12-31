package m

import (
	"reflect"
	"strconv"
	"time"
)

func timeFromAny(v any) time.Time {
	switch v := v.(type) {
	case string:
		t, _ := time.Parse(time.RFC3339, v)
		return t
	case time.Time:
		return v
	default:
		return time.Time{}
	}
}

type number interface {
	uint | uint64
}

func stringOrNull[T number](v *T) any {
	if v == nil {
		return nil
	}
	return strconv.Itoa(int(*v))
}

func anyToSlice(v any) []any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Slice {
		var result []any
		for i := 0; i < val.Len(); i++ {
			result = append(result, val.Index(i).Interface())
		}
		return result
	}
	return nil
}
