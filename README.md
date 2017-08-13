# JustSQL

JustSQL is a tool to generate golang wrapper code for SQL queries. It's inspired by [xo](https://github.com/knq/xo), but base on [TiDB](https://github.com/pingcap/tidb) to 'understand' SQL. Thus it only supports what TiDB supports: a majority of MySQL grammar (Also see: [Compatibility with MySQL](https://github.com/pingcap/docs/blob/master/sql/mysql-compatibility.md)). But since it directly invokes TiDB's parser/compiler to process SQL, it has more 'knowledge' to generate more friendly code.

### Features
- Single standalone executable, no need to connect to a real database. (It has an embedded one)
- **Just** feed it with normal DDL and DML **SQL** (and with annotations in comments), that's all. 
- Friendly code. See examples below.
- Custom code templates.

### Installation
#### Download precompiled binary
https://github.com/huangjunwen/JustSQL/releases
#### Compile from source
You should first download and compile tidb, then 
```
$ go get -u -v https://github.com/huangjunwen/JustSQL/justsql
```

### Quick starts
Create the following two sql files.
```
$ cat ddl.sql
CREATE TABLE user (
    id int AUTO_INCREMENT primary key,
    nick varchar(64) not null default '',
    gender enum('male', 'female', '') default null,
    tag set('a', 'b', 'c', '', 'd', 'x') default null,
    fill_time timestamp default current_timestamp
);

$ cat dml.sql
-- $func:AllUser
SELECT * FROM user;
```
Run command:
```
$ justsql -ddl ddl.sql -dml dml.sql -o model
```
That's it! There will be three files in `model` dir:
```
justsql.go // Some pkg level declarations.
user.tb.go // One for each table.
dml.sql.go // One for each dml file.
```
`dml.sql.go` will be like:
```
...
// AllUserResult is the return type of AllUser.
type AllUserResult struct {
    User *User
}

func newAllUserResult() *AllUserResult {
    return &AllUserResult{
        User: new(User),
    }
}

var _AllUserSQLTmpl = template.Must(template.New("AllUser").Parse("" +
    "SELECT user.id, user.nick, user.gender, user.tag, user.fill_time FROM user " + ""))

// AllUser is generated from:
//
//    -- $func:AllUser
//    SELECT * FROM user;
//
func AllUser(ctx_ context.Context, db_ DBer) ([]*AllUserResult, error) {

    // - Dot object for template and query parameter.
    dot_ := map[string]interface{}{}

    // - Render from template.
    buf_ := new(bytes.Buffer)
    if err_ := _AllUserSQLTmpl.Execute(buf_, dot_); err_ != nil {
    	...
    }
    ...
}

```

### TODO ....


