# structspec

Generate static results of struct reflection.

This tool generates special structs that contain all fields from the source
struct.

It allows mentioning some fields of source structs without the need to remember
the field names. This especially helps when you write queries to the database,
validating JSON/XML/etc fields.

## Installation

```shell
go install github.com/kazhuravlev/structspec@latest
```

## Usage

```shell
$ structspec gen --help
NAME:
   structspec gen - Generate and write result

USAGE:
   structspec gen [command options] [arguments...]

OPTIONS:
   --src value                          Source directory
   --structs value [ --structs value ]  Which structs should be included. Default: all founded
   --ignore value [ --ignore value ]    Which structs should be ignored. Default: no one
   --tag value [ --tag value ]          Which tags should be used for generation. Default: all founded
   --out-file value                     Output filename
   --out-pkg value                      Output package name
   --help, -h                           show help (default: false)

$ structspec gen \
  --src ./path/to/package/with/target/structs \
  --out-pkg mypackage
```

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
	query := fmt.Sprintf(
		queryTpl,
		structs.UserSpecData.Sql.ID,
		structs.UserSpecData.Sql.Username,
		structs.UserSpecData.Sql.ID,
	)
	fmt.Println(query)
	// select id, username from users where id = ?
}

```

This tool guarantees you never forget to update your queries after updating
source structs. All you need is to run `go generate` and the go compiler will
highlight what you forgot to update.
