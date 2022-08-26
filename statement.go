package mysqlparser_go

import (
	"fmt"
	"strings"
)

type MySQLStatement interface {
	Type() string
	Value() string
	GetFsmMap() []FsmMap
	ParseByFsm(fsmMap []FsmMap, tokenList MySQLTokenList, specialFinalStatus []int,
		verboseFunc func(message string, level LogLevel)) int
}

func GetStatementGen(t string) func(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	funcMap := map[string]func(tokenList MySQLTokenList,
		verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList){
		"AlterDatabaseStatement":  NewAlterDatabaseStatement,
		"AlterTableStatement":     NewAlterTableStatement,
		"CreateDatabaseStatement": NewCreateDatabaseStatement,
		"CreateTableStatement":    NewCreateTableStatement,
		"CreateIndexStatement":    NewCreateIndexStatement,
		"DeleteStatement":         NewDeleteStatement,
		"DropDatabaseStatement":   NewDropDatabaseStatement,
		"DropTableStatement":      NewDropTableStatement,
		"DropIndexStatement":      NewDropIndexStatement,
		"ExplainStatement":        NewExplainStatement,
		"InsertStatement":         NewInsertStatement,
		"RenameTableStatement":    NewRenameTableStatement,
		"ReplaceStatement":        NewReplaceStatement,
		"SelectStatement":         NewSelectStatement,
		"UnionStatement":          NewUnionStatement,
		"SetStatement":            NewSetStatement,
		"ShowStatement":           NewShowStatement,
		"TruncateTableStatement":  NewTruncateTableStatement,
		"UpdateStatement":         NewUpdateStatement,
		"UseStatement":            NewUseStatement,
	}
	return funcMap[t]
}

type MySQLBaseStatement struct {
	status     int
	value      string
	ObjectList []*MySQLObject
}

func (s *MySQLBaseStatement) Type() string {
	return "MySQLBaseStatement"
}

func (s *MySQLBaseStatement) Value() string {
	return s.value
}

func (s *MySQLBaseStatement) GetFsmMap() []FsmMap {
	fsmMap := make([]FsmMap, 0)
	return fsmMap
}

func (s *MySQLBaseStatement) ParseByFsm(fsmMap []FsmMap, tokenList MySQLTokenList, specialFinalStatus []int,
	verboseFunc func(message string, level LogLevel)) int {
	finalStatus := FinalStatus
	lastTermPos := tokenList.CurrentPos()
	lastTokenListPos, lastTermStatus := 0, 0
	for !tokenList.EOF() {
		t := tokenList.Next()
		if t == nil {
			break
		}
		if verboseFunc != nil {
			verboseFunc(fmt.Sprintf("NOW DEAL WITH %s: %s", (*t).Type(), (*t).Value()),
				LogLevelNotice)
		}
		if (*t).Type() == "MySQLCommentToken" || (*t).Type() == "MySQLSpaceToken" {
			obj := (*t).(MySQLObject)
			s.ObjectList = append(s.ObjectList, &obj)
			s.value += (*t).Value()
			continue
		} else if (*t).Type() == "MySQLDelimiterToken" && (*t).Value() == ";" {
			break
		}
		ruleFounded := false
		for _, rule := range fsmMap {
			statusMatch := InArray(s.status, rule.StartStatus)
			AcceptObjectType := GetObjectType(rule.AcceptObject)
			if statusMatch && AcceptObjectType == TOKEN {
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("MATCH TOKEN RULE %s", rule.ToString()),
						LogLevelNotice)
				}
				if (*t).Type() == rule.AcceptObject && (rule.AcceptValue == "" || (*t).Value() == rule.AcceptValue) {
					ruleFounded = true
					s.status = rule.EndStatus
					if verboseFunc != nil {
						verboseFunc(fmt.Sprintf("CHANGE STATUS TO %d", s.status),
							LogLevelNotice)
					}
					obj := (*t).(MySQLObject)
					s.ObjectList = append(s.ObjectList, &obj)
					s.value += (*t).Value()
					if s.status == finalStatus {
						if verboseFunc != nil {
							verboseFunc("STATUS END", LogLevelNotice)
						}
						return tokenList.CurrentPos()
					} else if InArray(s.status, specialFinalStatus) {
						lastTermPos = tokenList.CurrentPos()
						lastTermStatus = s.status
						lastTokenListPos = len(s.ObjectList)
						if verboseFunc != nil {
							verboseFunc(
								fmt.Sprintf("STATUS IN SPECIAL, SAVE POS %d, STATUS %d",
									lastTermPos, lastTermStatus),
								LogLevelNotice)
						}
					}
					break
				}
			} else if statusMatch &&
				(AcceptObjectType == COMPONENT || AcceptObjectType == STATEMENT) {
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("MATCH COMPLEX RULE %s", rule.ToString()),
						LogLevelNotice)
				}
				tokenList.Reset(tokenList.CurrentPos() - 1)
				var obj MySQLObject
				if AcceptObjectType == COMPONENT {
					obj, tokenList = GetComponentGen(rule.AcceptObject)(tokenList, verboseFunc)
				} else {
					obj, tokenList = GetStatementGen(rule.AcceptObject)(tokenList, verboseFunc)
				}
				if obj != nil {
					ruleFounded = true
					s.status = rule.EndStatus
					if verboseFunc != nil {
						verboseFunc(fmt.Sprintf("CHANGE STATUS TO %d", s.status),
							LogLevelNotice)
					}
					s.ObjectList = append(s.ObjectList, &obj)
					s.value += obj.Value()
					if s.status == finalStatus {
						if verboseFunc != nil {
							verboseFunc("STATUS END", LogLevelNotice)
						}
						return tokenList.CurrentPos()
					} else if InArray(s.status, specialFinalStatus) {
						lastTermPos = tokenList.CurrentPos()
						lastTermStatus = s.status
						lastTokenListPos = len(s.ObjectList)
						if verboseFunc != nil {
							verboseFunc(
								fmt.Sprintf("STATUS IN SPECIAL, SAVE POS %d, STATUS %d",
									lastTermPos, lastTermStatus), LogLevelNotice)
						}
					}
					break
				} else {
					t = tokenList.Next()
				}
			}
		}
		if !ruleFounded {
			if lastTermStatus > 0 {
				s.status = lastTermStatus
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("STATUS BACK TO %d", s.status), LogLevelNotice)
				}
				s.ObjectList = s.ObjectList[:lastTokenListPos]
				tmpList := make([]string, len(s.ObjectList))
				for index, tmpT := range s.ObjectList {
					tmpList[index] = (*tmpT).Value()
				}
				s.value = strings.Join(tmpList, "")
				tokenList.Reset(lastTermPos)
				return lastTermPos
			} else {
				tokenList.Reset(tokenList.CurrentPos() - 1)
				return -1
			}
		}
	}
	if InArray(s.status, specialFinalStatus) {
		return tokenList.CurrentPos()
	}
	return -1
}

// 13.1.1 ALTER DATABASE Syntax
// ALTER { DATABASE | SCHEMA } [db_name]
//   alter_specification ...
// ALTER { DATABASE | SCHEMA } db_name
//   UPGRADE DATA DIRECTORY NAME

type AlterDatabaseStatement struct {
	*MySQLBaseStatement
	Database string
}

func (s *AlterDatabaseStatement) Type() string {
	return "AlterDatabaseStatement"
}

func (s *AlterDatabaseStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALTER",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATABASE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SCHEMA",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLDatabaseNameComponent",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{2, 3, 7},
			AcceptObject: "MySQLDatabaseOptionComponent",
			AcceptValue:  "",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UPGRADE",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATA",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DIRECTORY",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NAME",
			EndStatus:    FinalStatus,
		},
	}
}

func NewAlterDatabaseStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &AlterDatabaseStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{7}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLDatabaseNameComponent" {
				s.Database = (*t).(*MySQLDatabaseNameComponent).Database
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.1.7 ALTER TABLE Syntax
// ALTER [ONLINE|OFFLINE] [IGNORE] TABLE tbl_name
//   [alter_specification [, alter_specification] ...]
//   [partition_options]

type AlterTableStatement struct {
	*MySQLBaseStatement
	Database string
	Table    string
}

func (s *AlterTableStatement) Type() string {
	return "AlterTableStatement"
}

func (s *AlterTableStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALTER",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ONLINE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OFFLINE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLE",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5, 7},
			AcceptObject: "MySQLAlterTableSpecificationComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{5, 6, 7},
			AcceptObject: "MySQLPartitionOptionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewAlterTableStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &AlterTableStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{5, 6}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.Database = (*t).(*MySQLTableNameComponent).Database
				s.Table = (*t).(*MySQLTableNameComponent).Table
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.1.10 CREATE DATABASE Syntax
// CREATE { DATABASE | SCHEMA } [IF NOT EXISTS] db_name
//   [create_specification] ...

type CreateDatabaseStatement struct {
	*MySQLBaseStatement
	Database string
}

func (s *CreateDatabaseStatement) Type() string {
	return "CreateDatabaseStatement"
}

func (s *CreateDatabaseStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CREATE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATABASE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SCHEMA",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IF",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NOT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EXISTS",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{2, 5},
			AcceptObject: "MySQLDatabaseNameComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6, 7},
			AcceptObject: "MySQLDatabaseOptionComponent",
			AcceptValue:  "",
			EndStatus:    7,
		},
	}
}

func NewCreateDatabaseStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &CreateDatabaseStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{6, 7}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLDatabaseNameComponent" {
				s.Database = (*t).(*MySQLDatabaseNameComponent).Database
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.1.17 CREATE TABLE Syntax
// CREATE [TEMPORARY] TABLE [IF NOT EXISTS] tbl_name
//   (create_specification ...)
//   [table_options]
//   [partition_options]
//
// CREATE [TEMPORARY] TABLE [IF NOT EXISTS] tbl_name
//   (create_specification ...)
//   [table_options]
//   [partition_options]
//   [IGNORE | REPLACE]
//   [AS] query_expression
//
// CREATE [TEMPORARY] TABLE [IF NOT EXISTS] tbl_name
//   { LIKE old_tbl_name | (LIKE old_tbl_name) }

type CreateTableStatement struct {
	*MySQLBaseStatement
	DatabaseList []string
	TableList    []string
	FromDatabase string
	FromTable    string
}

func (s *CreateTableStatement) Type() string {
	return "CreateTableStatement"
}

func (s *CreateTableStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CREATE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TEMPORARY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IF",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NOT",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EXISTS",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{3, 6},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8, 17},
			AcceptObject: "MySQLCreateTableDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLTableOptionListComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10, 11},
			AcceptObject: "MySQLPartitionOptionComponent",
			AcceptValue:  "",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{10, 11, 12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{10, 11, 12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REPLACE",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{10, 11, 12, 13},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AS",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{10, 11, 12, 13, 14},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIKE",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIKE",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
	}
}

func NewCreateTableStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &CreateTableStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{10, 11, 12}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				if len(s.TableList) > 0 {
					s.FromDatabase = (*t).(*MySQLTableNameComponent).Database
					s.FromTable = (*t).(*MySQLTableNameComponent).Table
				} else {
					s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
					s.TableList = append(s.TableList, (*t).(*MySQLTableNameComponent).Table)
				}
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						s.DatabaseList = append(s.DatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						s.TableList = append(s.TableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

//  13.1.13 CREATE INDEX Syntax
// CREATE [ONLINE|OFFLINE] [UNIQUE|FULLTEXT|SPATIAL] INDEX index_name
//    [index_type]
//    ON tbl_name (index_col_name,...)
//    [index_option] ...

type CreateIndexStatement struct {
	*MySQLBaseStatement
	Database string
	Table    string
}

func (s *CreateIndexStatement) Type() string {
	return "CreateIndexStatement"
}

func (s *CreateIndexStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CREATE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ONLINE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OFFLINE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNIQUE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FULLTEXT",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SPATIAL",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLIndexTypeComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{5, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ON",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLIndexColumnNameListComponent",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11, 12},
			AcceptObject: "MySQLIndexOptionComponent",
			AcceptValue:  "",
			EndStatus:    12,
		},
	}
}

func NewCreateIndexStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &CreateIndexStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{11, 12}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.Database = (*t).(*MySQLTableNameComponent).Database
				s.Table = (*t).(*MySQLTableNameComponent).Table
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.2.2 DELETE Syntax
// DELETE [LOW_PRIORITY] [QUICK] [IGNORE] FROM tbl_name
//    [WHERE where_condition]
//    [ORDER BY ...]
//    [LIMIT row_count]
//
// DELETE [LOW_PRIORITY] [QUICK] [IGNORE]
//    tbl_name[.*] [, tbl_name[.*]] ...
//    FROM table_references
//    [WHERE where_condition]
//
// DELETE [LOW_PRIORITY] [QUICK] [IGNORE]
//    FROM tbl_name[.*] [, tbl_name[.*]] ...
//    USING table_references
//    [WHERE where_condition]

type DeleteStatement struct {
	*MySQLBaseStatement
	DatabaseList []string
	TableList    []string
}

func (s *DeleteStatement) Type() string {
	return "DeleteStatement"
}

func (s *DeleteStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DELETE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LOW_PRIORITY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "QUICK",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{1, 2, 3, 4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WHERE",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{6, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ORDER",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLOrderListOptionComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{6, 8, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIMIT",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{6, 16},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ".",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "*",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{6, 14, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "USING",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{1, 2, 3, 4},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ".",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "*",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{20},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{20},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FROM",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "TableReferenceListComponent",
			AcceptValue:  "",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WHERE",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewDeleteStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &DeleteStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{6, 8, 11, 21}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
				s.TableList = append(s.TableList, (*t).(*MySQLTableNameComponent).Table)
			} else if (*t).Type() == "TableReferenceListComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*TableReferenceListComponent).DatabaseList...)
				s.TableList = append(s.TableList, (*t).(*TableReferenceListComponent).TableList...)
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						s.DatabaseList = append(s.DatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						s.TableList = append(s.TableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.1.21 DROP DATABASE Syntax
// DROP { DATABASE | SCHEMA } [IF EXISTS] db_name

type DropDatabaseStatement struct {
	*MySQLBaseStatement
	Database string
}

func (s *DropDatabaseStatement) Type() string {
	return "DropDatabaseStatement"
}

func (s *DropDatabaseStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DROP",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATABASE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SCHEMA",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IF",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EXISTS",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{2, 4},
			AcceptObject: "MySQLDatabaseNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewDropDatabaseStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &DropDatabaseStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLDatabaseNameComponent" {
				s.Database = (*t).(*MySQLDatabaseNameComponent).Database
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.1.28 DROP TABLE Syntax
// DROP [TEMPORARY] TABLE [IF EXISTS] tbl_name
//   tbl_name [, tbl_name] ...
//   [RESTRICT | CASCADE]

type DropTableStatement struct {
	*MySQLBaseStatement
	Database string
	Table    string
}

func (s *DropTableStatement) Type() string {
	return "DropTableStatement"
}

func (s *DropTableStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DROP",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TEMPORARY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IF",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EXISTS",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{3, 5},
			AcceptObject: "MySQLTableNameListComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RESTRICT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CASCADE",
			EndStatus:    FinalStatus,
		},
	}
}

func NewDropTableStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &DropTableStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{6}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.Database = (*t).(*MySQLTableNameComponent).Database
				s.Table = (*t).(*MySQLTableNameComponent).Table
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

//  13.1.24 DROP INDEX Syntax
// DROP [ONLINE|OFFLINE] INDEX index_name ON tbl_name

type DropIndexStatement struct {
	*MySQLBaseStatement
	Database string
	Table    string
}

func (s *DropIndexStatement) Type() string {
	return "DropIndexStatement"
}

func (s *DropIndexStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DROP",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ONLINE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OFFLINE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ON",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewDropIndexStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &DropIndexStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.Database = (*t).(*MySQLTableNameComponent).Database
				s.Table = (*t).(*MySQLTableNameComponent).Table
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.8.2 EXPLAIN Syntax
// {EXPLAIN | DESCRIBE | DESC}
//    tbl_name [col_name | wild]
//
// {EXPLAIN | DESCRIBE | DESC}
//    [explain_type] SELECT select_options
//
// explain_type: {EXTENDED | PARTITIONS}

type ExplainStatement struct {
	*MySQLBaseStatement
	DatabaseList []string
	TableList    []string
}

func (s *ExplainStatement) Type() string {
	return "ExplainStatement"
}

func (s *ExplainStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EXPLAIN",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESCRIBE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESC",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EXTENDED",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITIONS",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 3},
			AcceptObject: "UnionStatement",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1, 3},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewExplainStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &ExplainStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{2}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
				s.TableList = append(s.TableList, (*t).(*MySQLTableNameComponent).Table)
			} else if (*t).Type() == "SelectStatement" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*SelectStatement).DatabaseList...)
				s.TableList = append(s.TableList, (*t).(*SelectStatement).TableList...)
			} else if (*t).Type() == "UnionStatement" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*UnionStatement).DatabaseList...)
				s.TableList = append(s.TableList, (*t).(*UnionStatement).TableList...)
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.2.5 INSERT Syntax
// INSERT [LOW_PRIORITY | DELAYED | HIGH_PRIORITY] [IGNORE]
//    [INTO] tbl_name
//    [(col_name [, col_name] ...)]
//    {VALUES | VALUE} (value_list) [, (value_list)] ...
//    [ON DUPLICATE KEY UPDATE assignment_list]
//
// INSERT [LOW_PRIORITY | DELAYED | HIGH_PRIORITY] [IGNORE]
//    [INTO] tbl_name
//    SET assignment_list
//    [ON DUPLICATE KEY UPDATE assignment_list]
//
// INSERT [LOW_PRIORITY | HIGH_PRIORITY] [IGNORE]
//    [INTO] tbl_name
//    [(col_name [, col_name] ...)]
//    SELECT ...
//    [ON DUPLICATE KEY UPDATE assignment_list]
//
// value:
//    {expr | DEFAULT}
//
// value_list:
//    value [, value] ...
//

type InsertStatement struct {
	*MySQLBaseStatement
	DatabaseList     []string
	TableList        []string
	FromDatabaseList []string
	FromTableList    []string
}

func (s *InsertStatement) Type() string {
	return "InsertStatement"
}

func (s *InsertStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INSERT",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LOW_PRIORITY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "HIGH_PRIORITY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DELAYED",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INTO",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{1, 2, 3, 4},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLColumnNameListComponent",
			AcceptValue:  "",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VALUES",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VALUE",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLAssignmentListExpressionComponent",
			AcceptValue:  "",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "UnionStatement",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{12, 14, 15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ON",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DUPLICATE",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UPDATE",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLAssignmentListExpressionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewInsertStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &InsertStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList:     make([]string, 0),
		TableList:        make([]string, 0),
		FromDatabaseList: make([]string, 0),
		FromTableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{12, 14, 15}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
				s.TableList = append(s.TableList, (*t).(*MySQLTableNameComponent).Table)
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						s.FromDatabaseList = append(s.FromDatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						s.FromTableList = append(s.FromTableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			} else if (*t).Type() == "SelectStatement" {
				s.FromDatabaseList = append(s.FromDatabaseList, (*t).(*SelectStatement).DatabaseList...)
				s.FromTableList = append(s.FromTableList, (*t).(*SelectStatement).TableList...)
			} else if (*t).Type() == "UnionStatement" {
				s.FromDatabaseList = append(s.FromDatabaseList, (*t).(*UnionStatement).DatabaseList...)
				s.FromTableList = append(s.FromTableList, (*t).(*UnionStatement).TableList...)
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.1.32 RENAME TABLE Syntax
// RENAME TABLE
//   tbl_name TO new_tbl_name
//   [, tbl_name2 TO new_tbl_name2] ...

type RenameTableStatement struct {
	*MySQLBaseStatement
	DatabaseList     []string
	TableList        []string
	FromDatabaseList []string
	FromTableList    []string
}

func (s *RenameTableStatement) Type() string {
	return "RenameTableStatement"
}

func (s *RenameTableStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RENAME",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TO",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    2,
		},
	}
}

func NewRenameTableStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &RenameTableStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList:     make([]string, 0),
		TableList:        make([]string, 0),
		FromDatabaseList: make([]string, 0),
		FromTableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{5}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		i := 0
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				i += 1
				if i%2 == 1 {
					s.FromDatabaseList = append(s.FromDatabaseList, (*t).(*MySQLTableNameComponent).Database)
					s.FromTableList = append(s.FromTableList, (*t).(*MySQLTableNameComponent).Table)
				} else {
					s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
					s.TableList = append(s.TableList, (*t).(*MySQLTableNameComponent).Table)
				}
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.2.8 REPLACE Syntax
// REPLACE [LOW_PRIORITY | DELAYED] [IGNORE]
//    [INTO] tbl_name
//    [(col_name [, col_name] ...)]
//    {VALUES | VALUE} (value_list) [, (value_list)] ...
//
// REPLACE [LOW_PRIORITY | DELAYED] [IGNORE]
//    [INTO] tbl_name
//    SET assignment_list
//
// REPLACE [LOW_PRIORITY | DELAYED] [IGNORE]
//    [INTO] tbl_name
//    [(col_name [, col_name] ...)]
//    SELECT ...
//
// value:
//    {expr | DEFAULT}
//
// value_list:
//    value [, value] ...
//

type ReplaceStatement struct {
	*MySQLBaseStatement
	DatabaseList     []string
	TableList        []string
	FromDatabaseList []string
	FromTableList    []string
}

func (s *ReplaceStatement) Type() string {
	return "ReplaceStatement"
}

func (s *ReplaceStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REPLACE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LOW_PRIORITY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DELAYED",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INTO",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{1, 2, 3, 4},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLColumnNameListComponent",
			AcceptValue:  "",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VALUES",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VALUE",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLAssignmentListExpressionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "UnionStatement",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewReplaceStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &ReplaceStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList:     make([]string, 0),
		TableList:        make([]string, 0),
		FromDatabaseList: make([]string, 0),
		FromTableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{12}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
				s.TableList = append(s.TableList, (*t).(*MySQLTableNameComponent).Table)
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						s.FromDatabaseList = append(s.FromDatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						s.FromTableList = append(s.FromTableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			} else if (*t).Type() == "SelectStatement" {
				s.FromDatabaseList = append(s.FromDatabaseList, (*t).(*SelectStatement).DatabaseList...)
				s.FromTableList = append(s.FromTableList, (*t).(*SelectStatement).TableList...)
			} else if (*t).Type() == "UnionStatement" {
				s.FromDatabaseList = append(s.FromDatabaseList, (*t).(*UnionStatement).DatabaseList...)
				s.FromTableList = append(s.FromTableList, (*t).(*UnionStatement).TableList...)
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.2.9 SELECT Syntax
// SELECT
//    [ALL | DISTINCT | DISTINCTROW ]
//      [HIGH_PRIORITY]
//      [STRAIGHT_JOIN]
//      [SQL_SMALL_RESULT] [SQL_BIG_RESULT] [SQL_BUFFER_RESULT]
//      [SQL_CACHE | SQL_NO_CACHE] [SQL_CALC_FOUND_ROWS]
//    select_expr [, select_expr ...]
//    [FROM table_references
//    [WHERE where_condition]
//    [GROUP BY {col_name | expr | position}
//      [ASC | DESC], ... [WITH ROLLUP]]
//    [HAVING where_condition]
//    [ORDER BY {col_name | expr | position}
//      [ASC | DESC], ...]
//    [LIMIT {[offset,] row_count | row_count OFFSET offset}]
//    [PROCEDURE procedure_name(argument_list)]
//    [INTO OUTFILE 'file_name'
//        [CHARACTER SET charset_name]
//        export_options
//      | INTO DUMPFILE 'file_name'
//      | INTO var_name [, var_name]]
//    [FOR UPDATE | LOCK IN SHARE MODE]]

type SelectStatement struct {
	*MySQLBaseStatement
	DatabaseList []string
	TableList    []string
}

func (s *SelectStatement) Type() string {
	return "SelectStatement"
}

func (s *SelectStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SELECT",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALL",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISTINCT",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISTINCTROW",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "HIGH_PRIORITY",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STRAIGHT_JOIN",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{1, 2, 3, 4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SQL_SMALL_RESULT",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{1, 2, 3, 4, 5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SQL_BIG_RESULT",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{1, 2, 3, 4, 5, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SQL_BUFFER_RESULT",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{1, 2, 3, 4, 5, 6, 7},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SQL_CACHE",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{1, 2, 3, 4, 5, 6, 7},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SQL_NO_CACHE",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{1, 2, 3, 4, 5, 6, 7, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SQL_CALC_FOUND_ROWS",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 11},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AS",
			EndStatus:    51,
		},
		{
			StartStatus:  []int{51},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    52,
		},
		{
			StartStatus:  []int{51},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    52,
		},
		{
			StartStatus:  []int{10, 52},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10, 52},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FROM",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "TableReferenceListComponent",
			AcceptValue:  "",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WHERE",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{13, 15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "GROUP",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ASC",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESC",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WITH",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{20},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ROLLUP",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{13, 15, 18, 19, 21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "HAVING",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{13, 15, 18, 19, 21, 23},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ORDER",
			EndStatus:    24,
		},
		{
			StartStatus:  []int{24},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{25},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    26,
		},
		{
			StartStatus:  []int{25},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    26,
		},
		{
			StartStatus:  []int{25},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    26,
		},
		{
			StartStatus:  []int{26},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ASC",
			EndStatus:    27,
		},
		{
			StartStatus:  []int{26},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESC",
			EndStatus:    27,
		},
		{
			StartStatus:  []int{26, 27},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{13, 15, 18, 19, 21, 23, 26, 27},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIMIT",
			EndStatus:    28,
		},
		{
			StartStatus:  []int{28},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    29,
		},
		{
			StartStatus:  []int{29},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    30,
		},
		{
			StartStatus:  []int{29},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OFFSET",
			EndStatus:    30,
		},
		{
			StartStatus:  []int{30},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    31,
		},
		{
			StartStatus:  []int{13, 15, 18, 19, 21, 23, 26, 27, 29, 31},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PROCEDURE",
			EndStatus:    32,
		},
		{
			StartStatus:  []int{32},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    33,
		},
		{
			StartStatus:  []int{33},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    34,
		},
		{
			StartStatus:  []int{34},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    35,
		},
		{
			StartStatus:  []int{35},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    36,
		},
		{
			StartStatus:  []int{35},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    34,
		},
		{
			StartStatus:  []int{13, 15, 18, 19, 21, 23, 26, 27, 29, 31, 36},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INTO",
			EndStatus:    37,
		},
		{
			StartStatus:  []int{37},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OUTFILE",
			EndStatus:    38,
		},
		{
			StartStatus:  []int{38},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    39,
		},
		{
			StartStatus:  []int{39},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARACTER",
			EndStatus:    40,
		},
		{
			StartStatus:  []int{40},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    41,
		},
		{
			StartStatus:  []int{39},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARSET",
			EndStatus:    41,
		},
		{
			StartStatus:  []int{41},
			AcceptObject: "MySQLCharsetNameComponent",
			AcceptValue:  "",
			EndStatus:    42,
		},
		{
			StartStatus:  []int{39, 42},
			AcceptObject: "MySQLExportOptionComponent",
			AcceptValue:  "",
			EndStatus:    43,
		},
		{
			StartStatus:  []int{37},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DUMPFILE",
			EndStatus:    44,
		},
		{
			StartStatus:  []int{44},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    43,
		},
		{
			StartStatus:  []int{37, 46},
			AcceptObject: "MySQLVariableToken",
			AcceptValue:  "",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{45},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    46,
		},
		{
			StartStatus:  []int{13, 15, 18, 19, 21, 23, 26, 27, 29, 31, 36, 39, 42, 43, 45},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOR",
			EndStatus:    47,
		},
		{
			StartStatus:  []int{47},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UPDATE",
			EndStatus:    53,
		},
		{
			StartStatus:  []int{13, 15, 18, 19, 21, 23, 26, 27, 29, 31, 36, 39, 42, 43, 45},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LOCK",
			EndStatus:    48,
		},
		{
			StartStatus:  []int{48},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IN",
			EndStatus:    49,
		},
		{
			StartStatus:  []int{49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SHARE",
			EndStatus:    50,
		},
		{
			StartStatus:  []int{50},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MODE",
			EndStatus:    53,
		},
	}
}

func NewSelectStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &SelectStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList,
		[]int{10, 13, 15, 18, 19, 21, 23, 26, 27, 29, 31, 36, 39, 42, 43, 45, 52, 53}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "TableReferenceListComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*TableReferenceListComponent).DatabaseList...)
				s.TableList = append(s.TableList, (*t).(*TableReferenceListComponent).TableList...)
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						s.DatabaseList = append(s.DatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						s.TableList = append(s.TableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.2.9.3 UNION Syntax
// SELECT ...
// UNION [ALL | DISTINCT] SELECT ...
// [UNION [ALL | DISTINCT] SELECT ...]
//
// (SELECT a FROM t1 WHERE a=10 AND B=1 ORDER BY a LIMIT 10)
//  UNION
// (SELECT a FROM t2 WHERE a=11 AND B=2 ORDER BY a LIMIT 10);
//
// (SELECT a FROM t1 WHERE a=10 AND B=1)
//  UNION
// (SELECT a FROM t2 WHERE a=11 AND B=2)
//  ORDER BY a LIMIT 10;

type UnionStatement struct {
	*MySQLBaseStatement
	DatabaseList []string
	TableList    []string
}

func (s *UnionStatement) Type() string {
	return "UnionStatement"
}

func (s *UnionStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1, 4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNION",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALL",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISTINCT",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{2, 3},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7, 12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNION",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALL",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISTINCT",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{8, 9},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ORDER",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ASC",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESC",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{12, 15, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIMIT",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OFFSET",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewUnionStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &UnionStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{4, 12, 15, 16, 18}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "SelectStatement" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*SelectStatement).DatabaseList...)
				s.TableList = append(s.TableList, (*t).(*SelectStatement).TableList...)
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.7.4.1 SET Syntax for Variable Assignment
// SET variable_assignment [, variable_assignment] ...
//
// variable_assignment:
//      user_var_name = expr
//    | param_name = expr
//    | local_var_name = expr
//    | [GLOBAL | SESSION]
//        system_var_name = expr
//    | [@@global. | @@session. | @@]
//        system_var_name = expr
//
// SET ONE_SHOT system_var_name = expr
//
// 13.7.4.2 SET CHARACTER SET Syntax
// SET {CHARACTER SET | CHARSET}
//    {'charset_name' | DEFAULT}
//
// 13.7.4.3 SET NAMES Syntax
// SET NAMES {'charset_name'
//    [COLLATE 'collation_name'] | DEFAULT}

type SetStatement struct {
	*MySQLBaseStatement
}

func (s *SetStatement) Type() string {
	return "SetStatement"
}

func (s *SetStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "GLOBAL",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SESSION",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 6},
			AcceptObject: "MySQLVariableToken",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 6},
			AcceptObject: "MySQLUnquotedIdentifierToken",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ONE_SHOT",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLUnquotedIdentifierToken",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARACTER",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARSET",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLCharsetNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NAMES",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLCharsetNameComponent",
			AcceptValue:  "",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLLATE",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLCollationNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewSetStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &SetStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{5, 13}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.7.5 SHOW Syntax
// SHOW AUTHORS
// SHOW {BINARY | MASTER} LOGS
// SHOW BINLOG EVENTS [IN 'log_name'] [FROM pos] [LIMIT [offset,] row_count]
// SHOW CHARACTER SET [like_or_where]
// SHOW COLLATION [like_or_where]
// SHOW [FULL] {COLUMNS | FIELDS} {FROM | IN} tbl_name [{FROM | IN} db_name] [like_or_where]
// SHOW CONTRIBUTORS
// SHOW CREATE {DATABASE | SCHEMA} [IF NOT EXISTS] db_name
// SHOW CREATE EVENT event_name
// SHOW CREATE FUNCTION func_name
// SHOW CREATE PROCEDURE proc_name
// SHOW CREATE TABLE tbl_name
// SHOW CREATE TRIGGER trigger_name
// SHOW CREATE VIEW view_name
// SHOW {DATABASES | SCHEMAS} [like_or_where]
// SHOW ENGINE engine_name {STATUS | MUTEX}
// SHOW [STORAGE] ENGINES
// SHOW ERRORS [LIMIT [offset,] row_count]
// SHOW COUNT(*) ERRORS
// SHOW EVENTS [{FROM | IN} db_name} [like_or_where]
// SHOW FUNCTION CODE func_name
// SHOW FUNCTION STATUS [like_or_where]
// SHOW GRANTS [FOR user]
// SHOW {INDEX | INDEXES | KEYS} {FROM | IN} tbl_name [{FROM | IN} db_name] [like_or_where]
// SHOW MASTER STATUS
// SHOW OPEN TABLES [{FROM | IN} db_name] [like_or_where]
// SHOW PLUGINS
// SHOW PROCEDURE CODE proc_name
// SHOW PROCEDURE STATUS [like_or_where]
// SHOW PRIVILEGES
// SHOW [FULL] PROCESSLIST
// SHOW PROFILE [types] [FOR QUERY n] [LIMIT row_count [OFFSET offset]]
// SHOW PROFILES
// SHOW RELAYLOG EVENTS [IN 'log_name'] [FROM pos] [LIMIT [offset,] row_count]
// SHOW SLAVE HOSTS
// SHOW SLAVE STATUS
// SHOW [GLOBAL | SESSION] STATUS [like_or_where]
// SHOW TABLE STATUS [{FROM | IN} db_name] [like_or_where]
// SHOW [FULL] TABLES [{FROM | IN} db_name] [like_or_where]
// SHOW TRIGGERS [{FROM | IN} db_name] [like_or_where]
// SHOW [GLOBAL | SESSION] VARIABLES [like_or_where]
// SHOW WARNINGS [LIMIT [offset,] row_count]
// SHOW COUNT(*) WARNINGS
//
// like_or_where:
//    LIKE 'pattern'
//  | WHERE expr

type ShowStatement struct {
	*MySQLBaseStatement
	DatabaseList []string
	TableList    []string
}

func (s *ShowStatement) Type() string {
	return "ShowStatement"
}

func (s *ShowStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SHOW",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AUTHORS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BINARY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MASTER",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LOGS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BINLOG",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EVENTS",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IN",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{4, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FROM",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{4, 6, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIMIT",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARACTER",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARSET",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIKE",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{13, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WHERE",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLLATION",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FULL",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{1, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMNS",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{1, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FIELDS",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FROM",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IN",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FROM",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IN",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{20},
			AcceptObject: "MySQLDatabaseNameComponent",
			AcceptValue:  "",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CONTRIBUTORS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CREATE",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATABASE",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SCHEMA",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IF",
			EndStatus:    32,
		},
		{
			StartStatus:  []int{32},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NOT",
			EndStatus:    33,
		},
		{
			StartStatus:  []int{33},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EXISTS",
			EndStatus:    34,
		},
		{
			StartStatus:  []int{22, 34},
			AcceptObject: "MySQLDatabaseNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EVENT",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{23},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FUNCTION",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PROCEDURE",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLE",
			EndStatus:    24,
		},
		{
			StartStatus:  []int{24},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TRIGGER",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VIEW",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATABASES",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SCHEMAS",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENGINE",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{25},
			AcceptObject: "MySQLEngineNameComponent",
			AcceptValue:  "",
			EndStatus:    26,
		},
		{
			StartStatus:  []int{26, 42, 56},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STATUS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{26},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MUTEX",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STORAGE",
			EndStatus:    27,
		},
		{
			StartStatus:  []int{1, 27},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENGINES",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ERRORS",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COUNT",
			EndStatus:    35,
		},
		{
			StartStatus:  []int{35},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    36,
		},
		{
			StartStatus:  []int{36},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "*",
			EndStatus:    37,
		},
		{
			StartStatus:  []int{37},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    38,
		},
		{
			StartStatus:  []int{38},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ERRORS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "EVENTS",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FUNCTION",
			EndStatus:    28,
		},
		{
			StartStatus:  []int{28},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CODE",
			EndStatus:    29,
		},
		{
			StartStatus:  []int{29},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{28},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STATUS",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "GRANTS",
			EndStatus:    30,
		},
		{
			StartStatus:  []int{30},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOR",
			EndStatus:    31,
		},
		{
			StartStatus:  []int{31},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CURRENT_USER",
			EndStatus:    39,
		},
		{
			StartStatus:  []int{39},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    40,
		},
		{
			StartStatus:  []int{40},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{31},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    41,
		},
		{
			StartStatus:  []int{41},
			AcceptObject: "MySQLVariableToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEXES",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEYS",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MASTER",
			EndStatus:    42,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OPEN",
			EndStatus:    43,
		},
		{
			StartStatus:  []int{43},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLES",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PLUGINS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PRIVILEGES",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PROCEDURE",
			EndStatus:    28,
		},
		{
			StartStatus:  []int{1, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PROCESSLIST",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PROFILE",
			EndStatus:    44,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALL",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BLOCK",
			EndStatus:    46,
		},
		{
			StartStatus:  []int{46},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IO",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CONTEXT",
			EndStatus:    47,
		},
		{
			StartStatus:  []int{47},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SWITCHES",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CPU",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IPC",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MEMORY",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PAGE",
			EndStatus:    48,
		},
		{
			StartStatus:  []int{48},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FAULTS",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SOURCE",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{44, 49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SWAPS",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{45},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    49,
		},
		{
			StartStatus:  []int{44, 45},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOR",
			EndStatus:    50,
		},
		{
			StartStatus:  []int{50},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "QUERY",
			EndStatus:    51,
		},
		{
			StartStatus:  []int{51},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    52,
		},
		{
			StartStatus:  []int{44, 45, 52},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIMIT",
			EndStatus:    53,
		},
		{
			StartStatus:  []int{53},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    54,
		},
		{
			StartStatus:  []int{54},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OFFSET",
			EndStatus:    55,
		},
		{
			StartStatus:  []int{55},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PROFILES",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RELAYLOG",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SLAVE",
			EndStatus:    56,
		},
		{
			StartStatus:  []int{56},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "HOSTS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "GLOBAL",
			EndStatus:    57,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SESSION",
			EndStatus:    57,
		},
		{
			StartStatus:  []int{1, 57},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STATUS",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLE",
			EndStatus:    58,
		},
		{
			StartStatus:  []int{58},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STATUS",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{1, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLES",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TRIGGERS",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{1, 57},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VARIABLES",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WARNINGS",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{38},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WARNINGS",
			EndStatus:    FinalStatus,
		},
	}
}

func NewShowStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &ShowStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{4, 6, 8, 10, 13, 19, 30, 39, 44, 45, 52, 54}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLDatabaseNameComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLDatabaseNameComponent).Database)
				s.TableList = append(s.TableList, "")
			} else if (*t).Type() == "MySQLTableNameComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
				s.TableList = append(s.TableList, (*t).(*MySQLTableNameComponent).Table)
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						s.DatabaseList = append(s.DatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						s.TableList = append(s.TableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.1.33 TRUNCATE TABLE Syntax
// TRUNCATE [TABLE] tbl_name

type TruncateTableStatement struct {
	*MySQLBaseStatement
	Database string
	Table    string
}

func (s *TruncateTableStatement) Type() string {
	return "TruncateTableStatement"
}

func (s *TruncateTableStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TRUNCATE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLE",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewTruncateTableStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &TruncateTableStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				s.Database = (*t).(*MySQLTableNameComponent).Database
				s.Table = (*t).(*MySQLTableNameComponent).Table
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.2.11 UPDATE Syntax
// UPDATE [LOW_PRIORITY] [IGNORE] table_reference
//    SET assignment_list
//    [WHERE where_condition]
//    [ORDER BY ...]
//    [LIMIT row_count]
//
// UPDATE [LOW_PRIORITY] [IGNORE] table_references
//    SET assignment_list
//    [WHERE where_condition]
//
// value:
//    {expr | DEFAULT}

type UpdateStatement struct {
	*MySQLBaseStatement
	DatabaseList     []string
	TableList        []string
	FromDatabaseList []string
	FromTableList    []string
}

func (s *UpdateStatement) Type() string {
	return "UpdateStatement"
}

func (s *UpdateStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UPDATE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LOW_PRIORITY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 2, 3},
			AcceptObject: "TableReferenceListComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLAssignmentListExpressionComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WHERE",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{6, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ORDER",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ASC",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESC",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{6, 8, 11, 12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIMIT",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewUpdateStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &UpdateStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList:     make([]string, 0),
		TableList:        make([]string, 0),
		FromDatabaseList: make([]string, 0),
		FromTableList:    make([]string, 0),
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{6, 8, 11, 12}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "TableReferenceListComponent" {
				s.DatabaseList = append(s.DatabaseList, (*t).(*TableReferenceListComponent).DatabaseList...)
				s.TableList = append(s.TableList, (*t).(*TableReferenceListComponent).TableList...)
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						s.FromDatabaseList = append(s.FromDatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						s.FromTableList = append(s.FromTableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}

// 13.8.4 USE Syntax
// USE db_name

type UseStatement struct {
	*MySQLBaseStatement
	Database string
}

func (s *UseStatement) Type() string {
	return "UseStatement"
}

func (s *UseStatement) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "USE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDatabaseNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewUseStatement(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLStatement, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	s := &UseStatement{
		MySQLBaseStatement: &MySQLBaseStatement{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := s.ParseByFsm(s.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range s.ObjectList {
			if (*t).Type() == "MySQLDatabaseNameComponent" {
				s.Database = (*t).(*MySQLDatabaseNameComponent).Database
			}
		}
		tokenList.Reset(endPos)
		return s, tokenList
	}
}
