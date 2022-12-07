# structspec

Generate static results of struct reflection.

This tool generates special structs that contain all fields from the source
struct.

It allows mentioning some fields of source structs without the need to remember
the field names. This especially helps when you write queries to the database,
validating JSON/XML/etc fields.

## Example

Source struct:

```go
package structs

type User struct {
	ID       string `json:"id" sql:"user_id"`
	Username string `json:"username" sql:"username"`
}
```

Generated code:

```go
package structs

func (User) Spec() UserSpec {
	return UserSpecData
}

type UserSpec struct {
	Json UserSpecFields
	Sql  UserSpecFields
}

type UserSpecFields struct {
	ID       FieldName
	Username FieldName
}

var UserSpecData = UserSpec{
	Json: UserSpecFields{
		ID:       "id",
		Username: "username",
	},
	Sql: UserSpecFields{
		ID:       "user_id",
		Username: "username",
	},
}

// and helper functions ....

```

Usage in sql query builder:

```go
package storage

import "fmt"

func ExampleGetUserByID(id string) {
	queryTpl := `select %s, %s from users where %s = ?`
	query := fmt.Sprintf(queryTpl, structs.UserSpecData.Sql.ID, structs.UserSpecData.Sql.Username, structs.UserSpecData.Sql.ID)
	fmt.Println(query)
	// select id, username from users where id = ?
}
```

This tool guarantees you never forget to update your queries after updating
source structs. All you need is to run `go generate` and the go compiler will
highlight what you forgot to update.
