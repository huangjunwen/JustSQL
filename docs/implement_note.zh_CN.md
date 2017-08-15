### 一些实现中的要点

#### TableRefsMeta

- 强制要求 Table references 里每一个 Table source 都必须使用唯一的名字：即
```
SELECT * FROM booking b, (SELECT * FROM blog) b;
```
是不被允许的（MySQL 允许）

- 要求即使不同数据库里，表名如果相同的话则需要取别名，即
```
SELECT user.* FROM mysql.user, user;
```
需要改成。
```
SELECT user.* FROM mysql.user mu, user;
```
或的确需要两个表：
```
SELECT mu.*, user.* FROM mysql.user mu, user;
```

因以上两种都会让通配符产生歧义，**要保证 `SELECT tbl.* ... ` 是从单个 Table source (tbl) 取**。

参考：
- tidb/plan/resolver.go handleTableSource

#### FieldListMeta

- Field list 中一个带表名的通配符 (`tbl.*`) 对应一个 Table source 的全部列 (上述保证)
- Field list 中一个不带表名的通配符 (`*`) 对应所有 Table source 的全部列 
- 其余对应一个列

参考：
- tidb/plan/resolver.go createResultFields

#### 通配符展开

通配符展开需要每一个被展开的 field 都是有效的标识符：
```
SELECT tbl.* FROM (SELECT NOW()) tbl;
```
需改成
```
SELECT tbl.* FROM (SELECT NOW() now) tbl;
```
