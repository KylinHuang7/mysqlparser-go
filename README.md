# mysqlparser-go

`mysqlparser-go` is a golang lib for parsing [MySQL](http://dev.mysql.com) statements.

## Usage

```golang
package main

import (
	"fmt"
	mysqlparser "github.com/KylinHuang7/mysqlparser-go"
)

func main() {
	sql := "SELECT * FROM `dbData`.`tbTest` WHERE `iId` > 100 LIMIT 10"
	statementList, err := mysqlparser.Parse(sql)
	if err != nil {
		fmt.Println(err)
	}
	for _, statement := range statementList {
		fmt.Println(statement.Type())
		fmt.Println(statement.Value())
	}
}

```
