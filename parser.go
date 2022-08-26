package mysqlparser_go

import (
	"errors"
	"fmt"
)

func parseSingleSQL(tokenList MySQLTokenList) MySQLStatement {
	tokenStarts := tokenList.GetNextValidToken(2)
	if len(tokenStarts) != 2 {
		return nil
	}
	var s MySQLStatement
	switch (*tokenStarts[0]).Value() {
	case "CREATE":
		if InArray((*tokenStarts[1]).Value(), []string{"DATABASE", "SCHEMA"}) {
			s, tokenList = NewCreateDatabaseStatement(tokenList, verbose)
		} else if InArray((*tokenStarts[1]).Value(), []string{"TEMPORARY", "TABLE"}) {
			s, tokenList = NewCreateTableStatement(tokenList, verbose)
		} else if InArray((*tokenStarts[1]).Value(),
			[]string{"ONLINE", "OFFLINE", "UNIQUE", "FULLTEXT", "SPATIAL", "INDEX"}) {
			s, tokenList = NewCreateIndexStatement(tokenList, verbose)
		}
	case "ALTER":
		if InArray((*tokenStarts[1]).Value(), []string{"DATABASE", "SCHEMA"}) {
			s, tokenList = NewAlterDatabaseStatement(tokenList, verbose)
		} else if InArray((*tokenStarts[1]).Value(),
			[]string{"ONLINE", "OFFLINE", "IGNORE", "TABLE"}) {
			s, tokenList = NewAlterTableStatement(tokenList, verbose)
		}
	case "DROP":
		if InArray((*tokenStarts[1]).Value(), []string{"DATABASE", "SCHEMA"}) {
			s, tokenList = NewDropDatabaseStatement(tokenList, verbose)
		} else if InArray((*tokenStarts[1]).Value(), []string{"TEMPORARY", "TABLE"}) {
			s, tokenList = NewDropTableStatement(tokenList, verbose)
		} else if InArray((*tokenStarts[1]).Value(),
			[]string{"ONLINE", "OFFLINE", "INDEX"}) {
			s, tokenList = NewDropIndexStatement(tokenList, verbose)
		}
	case "RENAME":
		s, tokenList = NewRenameTableStatement(tokenList, verbose)
	case "TRUNCATE":
		s, tokenList = NewTruncateTableStatement(tokenList, verbose)
	case "SELECT", "(":
		if tokenList.HasToken("MySQLKeywordToken", "UNION") {
			s, tokenList = NewUnionStatement(tokenList, verbose)
		}
		if s == nil {
			s, tokenList = NewSelectStatement(tokenList, verbose)
		}
	case "INSERT":
		s, tokenList = NewInsertStatement(tokenList, verbose)
	case "REPLACE":
		s, tokenList = NewReplaceStatement(tokenList, verbose)
	case "UPDATE":
		s, tokenList = NewUpdateStatement(tokenList, verbose)
	case "DELETE":
		s, tokenList = NewDeleteStatement(tokenList, verbose)
	case "SET":
		s, tokenList = NewSetStatement(tokenList, verbose)
	case "SHOW":
		s, tokenList = NewShowStatement(tokenList, verbose)
	case "EXPLAIN", "DESCRIBE", "DESC":
		s, tokenList = NewExplainStatement(tokenList, verbose)
	case "USE":
		s, tokenList = NewUseStatement(tokenList, verbose)
	default:
		return nil
	}

	return s
}

func Parse(sql string) ([]MySQLStatement, error) {
	tokenList, err := NewMySQLTokenList(sql, verbose)
	if err != nil {
		return nil, err
	}
	verbose(fmt.Sprintf("Token List: %+v", tokenList), LogLevelInfo)

	sqlTokenList := tokenList.Divide()
	verbose(fmt.Sprintf("SQL Token List: %+v", sqlTokenList), LogLevelInfo)

	sqlList := make([]MySQLStatement, 0)
	for _, t := range sqlTokenList {
		s := parseSingleSQL(t)
		if s != nil {
			sqlList = append(sqlList, s)
			verbose(fmt.Sprintf("SQL Type: %s", s.Type()), LogLevelDebug)
		} else {
			verbose(fmt.Sprintf("SQL List: %+v", sqlList), LogLevelInfo)
			return nil, errors.New(fmt.Sprintf("Syntax error on  %+v", tokenList))
		}
	}

	return sqlList, nil
}
