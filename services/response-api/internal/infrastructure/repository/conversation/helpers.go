package conversation

import (
	"encoding/json"

	"gorm.io/datatypes"
)

func marshalJSON(value interface{}) (datatypes.JSON, error) {
	if value == nil {
		return datatypes.JSON([]byte("null")), nil
	}
	bytes, err := json.Marshal(value)
	return datatypes.JSON(bytes), err
}
