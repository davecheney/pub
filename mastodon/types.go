package mastodon

import (
	"fmt"

	"github.com/go-json-experiment/json"
)

// BoolOrBit is a type that can be unmarshalled from a JSON boolean or a JSON string
// iOS Ivory v13102+ sends "status": "1" or "status": "0" instead of "status": true or "status": false
// Credit to @diligiant for figuring this out and providing the code
type BoolOrBit bool

func (b *BoolOrBit) UnmarshalJSON(data []byte) error {
	var val any
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	switch v := val.(type) {
	case bool:
		*b = BoolOrBit(v)
	case float64:
		// 0 is false, != 0 is true
		*b = BoolOrBit(v != 0)
	case string:
		switch v {
		case "1", "true":
			*b = true
		case "0", "false":
			*b = false
		default:
			return fmt.Errorf("BoolOrBit unmarshal error: invalid input: %q", v)
		}
	default:
		return fmt.Errorf("BoolOrBit unmarshal error: invalid input: %q", v)
	}
	return nil
}
