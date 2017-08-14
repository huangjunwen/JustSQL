# JustSQL

JustSQL is a tool to generate golang wrapper code for SQL queries. It's inspired by [xo](https://github.com/knq/xo), but base on [TiDB](https://github.com/pingcap/tidb) to 'understand' SQL. Thus it only supports what TiDB supports: a majority of MySQL grammar (Also see: [Compatibility with MySQL](https://github.com/pingcap/docs/blob/master/sql/mysql-compatibility.md)). But since it directly invokes TiDB's parser/compiler to process SQL, it has more 'knowledge' to generate more friendly code.

### Features
- Single standalone executable, no need to connect to a real database. (It has an embedded one)
- **Just** feed it with normal DDL and DML **SQL** (and with annotations in comments), that's all. 
- Friendly code. See quick start below.
- Custom code templates.

### Installation
#### Download precompiled binary
https://github.com/huangjunwen/JustSQL/releases
#### Compile from source
You should first download and compile tidb, then 
```
$ go get -u -v github.com/huangjunwen/JustSQL/justsql
```

### Quick start
Let's show some example:
```sql
// $ cat ddl.sql

CREATE TABLE user (
    id INT AUTO_INCREMENT PRIMARY KEY,
    fill_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    nick VARCHAR(64) NOT NULL DEFAULT '',
    gender ENUM('male', 'female', '') DEFAULT NULL,
    tag SET('a', 'b', 'c', '', 'd', 'x') DEFAULT NULL
);

CREATE TABLE blog (
    id INT AUTO_INCREMENT PRIMARY KEY,
    fill_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_id INT NOT NULL,
    title VARCHAR(256) NOT NULL,
    content TEXT,
    FOREIGN KEY (user_id) REFERENCES user (id)
);

// $ cat dml.sql

-- $func:AllUser
SELECT * FROM user;

-- $func:QueryBlogById return:one
-- $arg:blogId type:int
SELECT b.*, u.nick
FROM blog b, user u
WHERE b.user_id=u.id AND b.id=/*$bind:blogId*/1/**/;

-- $func:QueryBlog return:many
-- $arg:userNick type:string
-- $arg:title type:string
SELECT *
FROM blog b JOIN user u ON (b.user_id=u.id)
WHERE 1
    /*$$ {{ if ne .userNick "" }}*/AND u.nick=/*$bind:userNick*/"jayven"/**/ /*$$ {{ end }}*/
    /*$$ {{ if ne .title "" }}*/AND b.title=/*$bind:title*/"How to use JustSQL?"/**/ /*$$ {{ end }}*/;
```
Run command:
```bash
$ justsql -ddl ddl.sql -dml dml.sql -o model
```
That's it! There will be four files generated in `model` directory:
```
justsql.go // Some pkg level declarations.
user.tb.go // One for each table.
blog.tb.go
dml.sql.go // One for each dml file.
```
`dml.sql.go` will be look like:
```golang
// --- AllUser generated code --- 
type AllUserResult struct {
    User *User
}

var _AllUserSQLTmpl = template.Must(template.New("AllUser").Parse("" +
    "SELECT user.id, user.fill_time, user.nick, user.gender, user.tag FROM user " + ""))

func AllUser(ctx_ context.Context, db_ DBer) ([]*AllUserResult, error) {
// ...
}

// --- QueryBlogById generated code --- 
type QueryBlogByIdResult struct {
    B    *Blog
    Nick string
}

var _QueryBlogByIdSQLTmpl = template.Must(template.New("QueryBlogById").Parse("" +
    "SELECT b.id, b.fill_time, b.user_id, b.title, b.content, u.nick " +
    "FROM blog b, user u " +
    "WHERE b.user_id=u.id AND b.id=:blogId " + ""))

func QueryBlogById(ctx_ context.Context, db_ DBer, blogId int) (*QueryBlogByIdResult, error) {
// ...
}

// --- QueryBlog generated code --- 
type QueryBlogResult struct {
    B *Blog
    U *User
}

var _QueryBlogSQLTmpl = template.Must(template.New("QueryBlog").Parse("" +
    "SELECT b.id, b.fill_time, b.user_id, b.title, b.content, u.nick " +
    "FROM blog b JOIN user u ON (b.user_id=u.id) " +
    "WHERE 1 " +
    "    {{ if ne .userNick \"\" }}AND u.nick=:userNick {{ end }} " +
    "    {{ if ne .title \"\" }}AND b.title=:title {{ end }} " + ""))

func QueryBlog(ctx_ context.Context, db_ DBer, userNick string, title string) ([]*QueryBlogResult, error) {
// ...
}
```
Some unique features are presented here:

- Wildcards in queries are automaticly expanded, this is more safer since table maybe altered in the future.
- Results of query are not just lists of return fields, wildcard of normal tables are grouped into nested struct for easier use.  



### Explanation

There is not much to say about `ddl.sql`, let's focus on `dml.sql`.

There are three SQL queries in `dml.sql`, also note that there are some special comments (so called _'annotations'_) before and inside the SQLs. Annotations are comments that having content starts with `$` to provide extra information about how to generate warpper code, or modification to the query. Here is the list:
| Name | Example | Usage |
|------|---------|-------|
| $func | $func:FuncName return:one | Declare a wrapper function and its return style: 'one' for single row and 'many' (default) for multiple rows |
| $arg | $arg:ArgName type:[]int | Declare a wrapper function argument and its type |
| $bind | $bind:BindName | Declare a named query binding, the content between the bind annotation and the next comment will be replace with `:BindName` (`:`  is configurable) |
| $$ ... | $$ Anything ... | Declare a block that will be substituted directly into the query text |

Combining the information extracted from SQL itself and the information from annotations, JustSQL is able to generate friendly code.

### LICENSE
MIT

### Author
huangjunwen (<kassarar@gmail.com>)


