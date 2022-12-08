package sample

import (
	"encoding/json"
	"github.com/kazhuravlev/structspec/internal/impl/testdata/structs"
)

type MyEntity struct {
	ID            structs.MyID    `pg:"id"`
	Name          Name            `pg:"name"`
	Age           int             `pg:"age"`
	IsOK          bool            `pg:"is_ok"`
	OptionalField *string         `pg:"optional_field"`
	JsonField     json.RawMessage `pg:"json_field"`
}

type Name string
