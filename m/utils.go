package m

import "strconv"

func utoa(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}
