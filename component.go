package mysqlparser_go

import (
	"fmt"
	"strings"
)

type MySQLComponent interface {
	Type() string
	Value() string
	GetFsmMap() []FsmMap
	ParseByFsm(fsmMap []FsmMap, tokenList MySQLTokenList, specialFinalStatus []int,
		verboseFunc func(message string, level LogLevel)) int
}

func GetComponentGen(t string) func(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	funcMap := map[string]func(tokenList MySQLTokenList,
		verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList){
		"MySQLIdentifierComponent":                NewMySQLIdentifierComponent,
		"MySQLDatabaseNameComponent":              NewMySQLDatabaseNameComponent,
		"MySQLTableNameComponent":                 NewMySQLTableNameComponent,
		"MySQLTableNameListComponent":             NewMySQLTableNameListComponent,
		"MySQLColumnNameComponent":                NewMySQLColumnNameComponent,
		"MySQLColumnNameListComponent":            NewMySQLColumnNameListComponent,
		"MySQLIndexColumnNameComponent":           NewMySQLIndexColumnNameComponent,
		"MySQLIndexColumnNameListComponent":       NewMySQLIndexColumnNameListComponent,
		"MySQLCharsetNameComponent":               NewMySQLCharsetNameComponent,
		"MySQLCollationNameComponent":             NewMySQLCollationNameComponent,
		"MySQLEngineNameComponent":                NewMySQLEngineNameComponent,
		"MySQLExpressionComponent":                NewMySQLExpressionComponent,
		"MySQLSubPartitioningExpressionComponent": NewMySQLSubPartitioningExpressionComponent,
		"MySQLPartitioningExpressionComponent":    NewMySQLPartitioningExpressionComponent,
		"MySQLAssignmentExpressionComponent":      NewMySQLAssignmentExpressionComponent,
		"MySQLAssignmentListExpressionComponent":  NewMySQLAssignmentListExpressionComponent,
		"MySQLNumericOptionValueComponent":        NewMySQLNumericOptionValueComponent,
		"MySQLBooleanOptionValueComponent":        NewMySQLBooleanOptionValueComponent,
		"MySQLStringOptionValueComponent":         NewMySQLStringOptionValueComponent,
		"MySQLDatabaseOptionComponent":            NewMySQLDatabaseOptionComponent,
		"MySQLTableOptionComponent":               NewMySQLTableOptionComponent,
		"MySQLTableOptionListComponent":           NewMySQLTableOptionListComponent,
		"MySQLIndexTypeComponent":                 NewMySQLIndexTypeComponent,
		"MySQLIndexOptionComponent":               NewMySQLIndexOptionComponent,
		"MySQLReferenceOptionComponent":           NewMySQLReferenceOptionComponent,
		"MySQLPartitionOptionComponent":           NewMySQLPartitionOptionComponent,
		"MySQLOrderOptionComponent":               NewMySQLOrderOptionComponent,
		"MySQLOrderListOptionComponent":           NewMySQLOrderListOptionComponent,
		"MySQLIndexHintOptionComponent":           NewMySQLIndexHintOptionComponent,
		"MySQLExportOptionComponent":              NewMySQLExportOptionComponent,
		"MySQLDataTypeComponent":                  NewMySQLDataTypeComponent,
		"MySQLReferenceDefinitionComponent":       NewMySQLReferenceDefinitionComponent,
		"MySQLColumnDefinitionComponent":          NewMySQLColumnDefinitionComponent,
		"MySQLSubPartitionDefinitionComponent":    NewMySQLSubPartitionDefinitionComponent,
		"MySQLPartitionDefinitionComponent":       NewMySQLPartitionDefinitionComponent,
		"MySQLCreateTableDefinitionComponent":     NewMySQLCreateTableDefinitionComponent,
		"SubQueryComponent":                       NewSubQueryComponent,
		"TableFactorComponent":                    NewTableFactorComponent,
		"TableReferenceComponent":                 NewTableReferenceComponent,
		"TableReferenceListComponent":             NewTableReferenceListComponent,
		"MySQLAlterTableSpecificationComponent":   NewMySQLAlterTableSpecificationComponent,
	}
	return funcMap[t]
}

type MySQLBaseComponent struct {
	status     int
	value      string
	ObjectList []*MySQLObject
}

func (c *MySQLBaseComponent) Type() string {
	return "MySQLBaseComponent"
}

func (c *MySQLBaseComponent) Value() string {
	return c.value
}

func (c *MySQLBaseComponent) GetFsmMap() []FsmMap {
	fsmMap := make([]FsmMap, 0)
	return fsmMap
}

func (c *MySQLBaseComponent) ParseByFsm(fsmMap []FsmMap, tokenList MySQLTokenList, specialFinalStatus []int,
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
			verboseFunc(fmt.Sprintf("COMPONENT NOW DEAL WITH %s: %s", (*t).Type(), (*t).Value()),
				LogLevelInfo)
		}
		if (*t).Type() == "MySQLCommentToken" || (*t).Type() == "MySQLSpaceToken" {
			obj := (*t).(MySQLObject)
			c.ObjectList = append(c.ObjectList, &obj)
			c.value += (*t).Value()
			continue
		} else if (*t).Type() == "MySQLDelimiterToken" && (*t).Value() == ";" {
			break
		}
		ruleFounded := false
		for _, rule := range fsmMap {
			statusMatch := InArray(c.status, rule.StartStatus)
			AcceptObjectType := GetObjectType(rule.AcceptObject)
			if statusMatch && AcceptObjectType == TOKEN {
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("COMPONENT MATCH TOKEN RULE %s", rule.ToString()),
						LogLevelInfo)
				}
				if (*t).Type() == rule.AcceptObject && (rule.AcceptValue == "" || (*t).Value() == rule.AcceptValue) {
					ruleFounded = true
					c.status = rule.EndStatus
					if verboseFunc != nil {
						verboseFunc(fmt.Sprintf("COMPONENT CHANGE STATUS TO %d", c.status),
							LogLevelInfo)
					}
					obj := (*t).(MySQLObject)
					c.ObjectList = append(c.ObjectList, &obj)
					c.value += (*t).Value()
					if c.status == finalStatus {
						if verboseFunc != nil {
							verboseFunc("COMPONENT STATUS END", LogLevelInfo)
						}
						return tokenList.CurrentPos()
					} else if InArray(c.status, specialFinalStatus) {
						lastTermPos = tokenList.CurrentPos()
						lastTermStatus = c.status
						lastTokenListPos = len(c.ObjectList)
						if verboseFunc != nil {
							verboseFunc(
								fmt.Sprintf("COMPONENT STATUS IN SPECIAL, SAVE POS %d, STATUS %d",
									lastTermPos, lastTermStatus),
								LogLevelInfo)
						}
					}
					break
				}
			} else if statusMatch &&
				(AcceptObjectType == COMPONENT || AcceptObjectType == STATEMENT) {
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("COMPONENT MATCH COMPLEX RULE %s", rule.ToString()),
						LogLevelInfo)
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
					c.status = rule.EndStatus
					if verboseFunc != nil {
						verboseFunc(fmt.Sprintf("COMPONENT CHANGE STATUS TO %d", c.status),
							LogLevelInfo)
					}
					c.ObjectList = append(c.ObjectList, &obj)
					c.value += obj.Value()
					if c.status == finalStatus {
						if verboseFunc != nil {
							verboseFunc("COMPONENT STATUS END", LogLevelInfo)
						}
						return tokenList.CurrentPos()
					} else if InArray(c.status, specialFinalStatus) {
						lastTermPos = tokenList.CurrentPos()
						lastTermStatus = c.status
						lastTokenListPos = len(c.ObjectList)
						if verboseFunc != nil {
							verboseFunc(
								fmt.Sprintf("COMPONENT STATUS IN SPECIAL, SAVE POS %d, STATUS %d",
									lastTermPos, lastTermStatus), LogLevelInfo)
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
				c.status = lastTermStatus
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("COMPONENT STATUS BACK TO %d", c.status), LogLevelInfo)
				}
				c.ObjectList = c.ObjectList[:lastTokenListPos]
				tmpList := make([]string, len(c.ObjectList))
				for index, tmpT := range c.ObjectList {
					tmpList[index] = (*tmpT).Value()
				}
				c.value = strings.Join(tmpList, "")
				tokenList.Reset(lastTermPos)
				return lastTermPos
			} else {
				tokenList.Reset(tokenList.CurrentPos() - 1)
				return -1
			}
		}
	}
	if InArray(c.status, specialFinalStatus) {
		return tokenList.CurrentPos()
	}
	return -1
}

// data_type:
//    BIT[(length)]
//  | TINYINT[(length)] [UNSIGNED] [ZEROFILL]
//  | SMALLINT[(length)] [UNSIGNED] [ZEROFILL]
//  | MEDIUMINT[(length)] [UNSIGNED] [ZEROFILL]
//  | INT[(length)] [UNSIGNED] [ZEROFILL]
//  | INTEGER[(length)] [UNSIGNED] [ZEROFILL]
//  | BIGINT[(length)] [UNSIGNED] [ZEROFILL]
//  | REAL[(length,decimals)] [UNSIGNED] [ZEROFILL]
//  | DOUBLE[(length,decimals)] [UNSIGNED] [ZEROFILL]
//  | FLOAT[(length,decimals)] [UNSIGNED] [ZEROFILL]
//  | DECIMAL[(length[,decimals])] [UNSIGNED] [ZEROFILL]
//  | NUMERIC[(length[,decimals])] [UNSIGNED] [ZEROFILL]
//  | DATE
//  | TIME
//  | TIMESTAMP
//  | DATETIME
//  | YEAR
//  | CHAR[(length)]
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | VARCHAR(length)
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | BINARY[(length)]
//  | VARBINARY(length)
//  | TINYBLOB
//  | BLOB[(length)]
//  | MEDIUMBLOB
//  | LONGBLOB
//  | TINYTEXT
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | TEXT[(length)]
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | MEDIUMTEXT
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | LONGTEXT
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | ENUM(value1,value2,value3,...)
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | SET(value1,value2,value3,...)
//      [CHARACTER SET charset_name] [COLLATE collation_name]
//  | spatial_type

type MySQLDataTypeComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLDataTypeComponent) Type() string {
	return "MySQLDataTypeComponent"
}

func (c *MySQLDataTypeComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATE",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TIME",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TIMESTAMP",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATETIME",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "YEAR",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TINYBLOB",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MEDIUMBLOB",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LONGBLOB",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BIT",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BINARY",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BLOB",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TINYINT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SMALLINT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MEDIUMINT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INTEGER",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BIGINT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6, 13, 16, 18},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{4, 7},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNSIGNED",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{4, 7, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ZEROFILL",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REAL",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DOUBLE",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FLOAT",
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
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DECIMAL",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NUMERIC",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TINYTEXT",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MEDIUMTEXT",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LONGTEXT",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19, 24},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARACTER",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{20},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{19, 24},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARSET",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLCharsetNameComponent",
			AcceptValue:  "",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{19, 22, 24},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLLATE",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{23},
			AcceptObject: "MySQLCollationNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHAR",
			EndStatus:    24,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TEXT",
			EndStatus:    24,
		},
		{
			StartStatus:  []int{24},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{25},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    26,
		},
		{
			StartStatus:  []int{26, 30},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VARCHAR",
			EndStatus:    27,
		},
		{
			StartStatus:  []int{27},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENUM",
			EndStatus:    28,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    28,
		},
		{
			StartStatus:  []int{28},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    29,
		},
		{
			StartStatus:  []int{29, 31},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    30,
		},
		{
			StartStatus:  []int{30},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    31,
		},
	}
}

func NewMySQLDataTypeComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLDataTypeComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1, 4, 7, 8, 9, 14, 19, 22, 24}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// reference_definition:
//    REFERENCES tbl_name (index_col_name,...)
//      [MATCH FULL | MATCH PARTIAL | MATCH SIMPLE]
//      [ON DELETE reference_option]
//      [ON UPDATE reference_option]

type MySQLReferenceDefinitionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLReferenceDefinitionComponent) Type() string {
	return "MySQLReferenceDefinitionComponent"
}

func (c *MySQLReferenceDefinitionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REFERENCES",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLIndexColumnNameListComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MATCH",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FULL",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTIAL",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SIMPLE",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{5, 7},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ON",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DELETE",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLReferenceOptionComponent",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ON",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{8, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UPDATE",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLReferenceOptionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLReferenceDefinitionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLReferenceDefinitionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{5, 7, 10}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// column_definition:
//    data_type [NOT NULL | NULL] [DEFAULT default_value]
//      [AUTO_INCREMENT | ON UPDATE CURRENT_TIMESTAMP] [UNIQUE [KEY]] [[PRIMARY] KEY]
//      [COMMENT 'string']
//      [COLUMN_FORMAT {FIXED|DYNAMIC|DEFAULT}]
//      [STORAGE {DISK|MEMORY|DEFAULT}]
//      [reference_definition]

type MySQLColumnDefinitionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLColumnDefinitionComponent) Type() string {
	return "MySQLColumnDefinitionComponent"
}

func (c *MySQLColumnDefinitionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLDataTypeComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1, 5, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NOT",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLNullToken",
			AcceptValue:  "NULL",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 3, 5, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLNullToken",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CURRENT_TIMESTAMP",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{1, 3, 5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ON",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UPDATE",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CURRENT_TIMESTAMP",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{1, 3, 5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AUTO_INCREMENT",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{1, 3, 5, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNIQUE",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{1, 3, 5, 6, 7, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PRIMARY",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{1, 9},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{3, 5, 6, 7, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{1, 3, 5, 6, 7, 8, 10},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COMMENT",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{1, 3, 5, 6, 7, 8, 10, 12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMN_FORMAT",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FIXED",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DYNAMIC",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{1, 3, 5, 6, 7, 8, 10, 12, 14},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STORAGE",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISK",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MEMORY",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{1, 3, 5, 6, 7, 8, 10, 12, 14, 16},
			AcceptObject: "MySQLReferenceDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLColumnDefinitionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLColumnDefinitionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1, 3, 5, 6, 7, 8, 10, 12, 14, 16}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// subpartition_definition:
//    SUBPARTITION logical_name
//        [[STORAGE] ENGINE [=] engine_name]
//        [COMMENT [=] 'string' ]
//        [DATA DIRECTORY [=] 'data_dir']
//        [INDEX DIRECTORY [=] 'index_dir']
//        [MAX_ROWS [=] max_number_of_rows]
//        [MIN_ROWS [=] min_number_of_rows]
//        [TABLESPACE [=] tablespace_name]
//        [NODEGROUP [=] node_group_id]

type MySQLSubPartitionDefinitionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLSubPartitionDefinitionComponent) Type() string {
	return "MySQLSubPartitionDefinitionComponent"
}

func (c *MySQLSubPartitionDefinitionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SUBPARTITION",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STORAGE",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENGINE",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{4, 5},
			AcceptObject: "MySQLEngineNameComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{2, 6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COMMENT",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{2, 6, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATA",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DIRECTORY",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{2, 6, 8, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DIRECTORY",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{2, 6, 8, 11, 14},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MAX_ROWS",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{2, 6, 8, 11, 14, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MIN_ROWS",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{2, 6, 8, 11, 14, 16, 18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLESPACE",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{19, 20},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{2, 6, 8, 11, 14, 16, 18, 21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NODEGROUP",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLSubPartitionDefinitionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLSubPartitionDefinitionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{2, 6, 8, 11, 14, 16, 18, 21}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// partition_definition:
//    PARTITION partition_name
//        [VALUES
//            {LESS THAN {(expr | value_list) | MAXVALUE}
//            |
//            IN (value_list)}]
//        [[STORAGE] ENGINE [=] engine_name]
//        [COMMENT [=] 'string' ]
//        [DATA DIRECTORY [=] 'data_dir']
//        [INDEX DIRECTORY [=] 'index_dir']
//        [MAX_ROWS [=] max_number_of_rows]
//        [MIN_ROWS [=] min_number_of_rows]
//        [TABLESPACE [=] tablespace_name]
//        [NODEGROUP [=] node_group_id]
//        [(subpartition_definition [, subpartition_definition] ...)]

type MySQLPartitionDefinitionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLPartitionDefinitionComponent) Type() string {
	return "MySQLPartitionDefinitionComponent"
}

func (c *MySQLPartitionDefinitionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITION",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "VALUES",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LESS",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "THAN",
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
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MAXVALUE",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7, 11},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MAXVALUE",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IN",
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
			AcceptObject: "MySQLNumericToken",
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
			StartStatus:  []int{2, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STORAGE",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{2, 8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENGINE",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENGINE",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{13, 14},
			AcceptObject: "MySQLEngineNameComponent",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{2, 8, 15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COMMENT",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{2, 8, 15, 17},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATA",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DIRECTORY",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{2, 8, 15, 17, 20},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DIRECTORY",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{2, 8, 15, 17, 20, 23},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MAX_ROWS",
			EndStatus:    24,
		},
		{
			StartStatus:  []int{24},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{2, 8, 15, 17, 20, 23, 25},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MIN_ROWS",
			EndStatus:    26,
		},
		{
			StartStatus:  []int{26},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    27,
		},
		{
			StartStatus:  []int{2, 8, 15, 17, 20, 23, 25, 27},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLESPACE",
			EndStatus:    28,
		},
		{
			StartStatus:  []int{28},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    29,
		},
		{
			StartStatus:  []int{28, 29},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    30,
		},
		{
			StartStatus:  []int{2, 8, 15, 17, 20, 23, 25, 27, 30},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NODEGROUP",
			EndStatus:    31,
		},
		{
			StartStatus:  []int{31},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    32,
		},
		{
			StartStatus:  []int{2, 8, 15, 17, 20, 23, 25, 27, 30, 32, 8, 15, 17, 20, 23, 25, 27, 30},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    33,
		},
		{
			StartStatus:  []int{33},
			AcceptObject: "MySQLSubPartitionDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    34,
		},
		{
			StartStatus:  []int{34},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{34},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    33,
		},
	}
}

func NewMySQLPartitionDefinitionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLPartitionDefinitionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{2, 8, 15, 17, 20, 23, 25, 27, 30, 32}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// create_definition:
//    col_name column_definition
//  | [CONSTRAINT [symbol]] PRIMARY KEY [index_type] (index_col_name,...)
//      [index_option] ...
//  | {INDEX|KEY} [index_name] [index_type] (index_col_name,...)
//      [index_option] ...
//  | [CONSTRAINT [symbol]] UNIQUE [INDEX|KEY]
//      [index_name] [index_type] (index_col_name,...)
//      [index_option] ...
//  | {FULLTEXT|SPATIAL} [INDEX|KEY] [index_name] (index_col_name,...)
//      [index_option] ...
//  | [CONSTRAINT [symbol]] FOREIGN KEY
//      [index_name] (index_col_name,...) reference_definition
//  | CHECK (expr)

type MySQLCreateTableDefinitionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLCreateTableDefinitionComponent) Type() string {
	return "MySQLCreateTableDefinitionComponent"
}

func (c *MySQLCreateTableDefinitionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLColumnDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CONSTRAINT",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{0, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PRIMARY",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5, 10, 11},
			AcceptObject: "MySQLIndexTypeComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{5, 6, 10, 11},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLIndexColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLIndexOptionComponent",
			AcceptValue:  "",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{0, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{0, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10, 11},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{0, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNIQUE",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FULLTEXT",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SPATIAL",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{0, 2, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOREIGN",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{13, 14},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLIndexColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLReferenceDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHECK",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLPartitioningExpressionComponent",
			AcceptValue:  "",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{20},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLCreateTableDefinitionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLCreateTableDefinitionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{9}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLExpressionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLExpressionComponent) Type() string {
	return "MySQLExpressionComponent"
}

var (
	supportKeyword = []string{
		"AND", "BETWEEN", "BINARY", "CASE", "COLLATE",
		"CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP", "CURRENT_USER", "DIV",
		"ELSE", "END", "EXISTS", "IN", "INTERVAL",
		"IS", "LAST_DAY", "LIKE", "LOCALTIME", "LOCALTIMESTAMP",
		"MATCH", "MOD", "NOT", "OR", "REGEXP",
		"RLIKE", "SOUNDS", "THEN", "WHEN", "XOR",
	}
	supportInFunctionKeyword = []string{
		"AS", "ASC", "BY", "DESC", "DISTINCT",
		"GROUP", "ORDER", "SEPARATOR", "USING",
	}
	supportFunction = []string{
		"ABS", "ACOS", "ADDDATE", "ADDTIME", "AES_DECRYPT",
		"AES_ENCRYPT", "ASCII", "ASIN", "ATAN", "ATAN2",
		"AVG", "BENCHMARK", "BIN", "BIT_AND", "BIT_COUNT",
		"BIT_LENGTH", "BIT_OR", "BIT_XOR", "CAST", "CEIL",
		"CEILING", "CHAR", "CHARACTER_LENGTH", "CHARSET", "CHAR_LENGTH",
		"COERCIBILITY", "COLLATION", "COMPRESS", "CONCAT", "CONCAT_WS",
		"CONNECTION_ID", "CONV", "CONVERT", "CONVERT_TZ", "COS",
		"COT", "COUNT", "CRC32", "CURDATE", "CURRENT_DATE",
		"CURRENT_TIME", "CURRENT_TIMESTAMP", "CURRENT_USER", "CURTIME", "DATABASE",
		"DATE", "DATEDIFF", "DATE_ADD", "DATE_FORMAT", "DATE_SUB",
		"DAY", "DAYNAME", "DAYOFMONTH", "DAYOFWEEK", "DAYOFYEAR",
		"DECODE", "DEFAULT", "DEGREES", "DES_DECRYPT", "DES_ENCRYPT",
		"ELT", "ENCODE", "ENCRYPT", "EXP", "EXPORT_SET",
		"EXTRACT", "FIELD", "FIND_IN_SET", "FLOOR", "FORMAT",
		"FORM_UNIXTIME", "FOUND_ROWS", "FROM_DAYS", "GET_FORMAT", "GET_LOCK",
		"GROUP_CONCAT", "HEX", "HOUR", "IF", "IFNULL",
		"INET_ATON", "INET_NTOA", "INSERT", "INSTR", "IS_FREE_LOCK",
		"IS_USED_LOCK", "LAST_INSERT_ID", "LCASE", "LEFT", "LENGTH",
		"LN", "LOAD_FILE", "LOCALTIME", "LOCALTIMESTAMP", "LOCATE",
		"LOG", "LOG10", "LOG2", "LOWER", "LPAD",
		"LTRIM", "MAKEDATE", "MAKETIME", "MAKE_SET", "MASTER_POS_WAIT",
		"MAX", "MD5", "MICROSECOND", "MID", "MIN",
		"MINUTE", "MOD", "MONTH", "MONTHNAME", "NAME_CONST",
		"NOW", "NULLIF", "OCT", "OCTET_LENGTH", "OLD_PASSWORD",
		"ORD", "PASSWORD", "PERIOD_ADD", "PERIOD_DIFF", "PI",
		"POSITION", "POW", "POWER", "QUARTER", "QUOTE",
		"RADIANS", "RAND", "RELEASE_LOCK", "REPEAT", "REPLACE",
		"REVERSE", "RIGHT", "ROUND", "ROW_COUNT", "RPAD",
		"RTRIM", "SCHEMA", "SECOND", "SESSION_USER", "SET_TO_TIME",
		"SHA", "SHA1", "SHA2", "SIGN", "SIN",
		"SLEEP", "SOUNDEX", "SPACE", "SQRT", "STD",
		"STDDEV", "STDDEV_POP", "STDDEV_SAMP", "STRCMP", "STR_TO_DATE",
		"SUBDATE", "SUBSTR", "SUBSTRING", "SUBSTRING_INDEX", "SUM",
		"SYSDATE", "SYSTEM_USER", "TAN", "TIME", "TIMEDIFF",
		"TIMESTAMP", "TIMESTAMPADD", "TIMESTAMPDIFF", "TIME_FORMAT", "TIME_TO_SEC",
		"TO_DAYS", "TO_SECONDS", "TRIM", "TRUNCATE", "UCASE",
		"UNCOMPRESS", "UNCOMPRESSED_LENGTH", "UNHEX", "UNIX_TIMESTAMP", "UPPER",
		"USER", "UTC_DATE", "UTC_TIME", "UTC_TIMESTAMP", "UUID",
		"UUID_SHORT", "VALUES", "VARIANCE", "VAR_POP", "VAR_SAMP",
		"VERSION", "WEEK", "WEEKDAY", "WEEKOFYEAR", "YEAR",
		"YEARWEEK",
	}
)

func NewMySQLExpressionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLExpressionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	inBracket := 0
	bracketTokenList := make([]MySQLToken, 0)
	bracketValue := ""
	lastTermPos := tokenList.CurrentPos()
	for !tokenList.EOF() {
		t := tokenList.Next()
		if t == nil {
			break
		}
		if verboseFunc != nil {
			verboseFunc(fmt.Sprintf("EXPRESSION DEAL WITH %s: %s", (*t).Type(), (*t).Value()),
				LogLevelInfo)
		}
		if (*t).Type() == "MySQLDelimiterToken" && (*t).Value() == ";" {
			tokenList.Reset(tokenList.CurrentPos() - 1)
			break
		} else if (*t).Type() == "MySQLDelimiterToken" && (*t).Value() == "," {
			if inBracket == 0 {
				tokenList.Reset(tokenList.CurrentPos() - 1)
				break
			} else {
				bracketTokenList = append(bracketTokenList, *t)
				bracketValue += (*t).Value()
				lastTermPos = tokenList.CurrentPos()
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("EXPRESSION IN ',' %+v, %s, %d, %d",
						bracketTokenList, bracketValue, lastTermPos, inBracket), LogLevelInfo)
				}
			}
		} else if (*t).Type() == "MySQLOperatorToken" && (*t).Value() == "(" {
			nextToken := tokenList.GetNextValidToken(1)
			if verboseFunc != nil {
				verboseFunc(fmt.Sprintf("EXPRESSION IN '(' %s: %s", (*t).Type(), (*t).Value()),
					LogLevelInfo)
			}
			if (*nextToken[0]).Type() == "MySQLKeywordToken" && (*nextToken[0]).Value() == "SELECT" {
				tokenList.Reset(tokenList.CurrentPos() - 1)
				var subQuery MySQLComponent
				subQuery, tokenList = NewSubQueryComponent(tokenList, verboseFunc)
				if subQuery != nil {
					if inBracket > 0 {
						bracketTokenList = append(bracketTokenList, subQuery)
						bracketValue += subQuery.Value()
						if verboseFunc != nil {
							verboseFunc(fmt.Sprintf("EXPRESSION IN SUBQUERY %+v, %s, %d, %d",
								bracketTokenList, bracketValue, lastTermPos, inBracket), LogLevelInfo)
						}
					} else {
						obj := subQuery.(MySQLObject)
						c.ObjectList = append(c.ObjectList, &obj)
						c.value += subQuery.Value()
						lastTermPos = tokenList.CurrentPos()
						if verboseFunc != nil {
							verboseFunc(fmt.Sprintf("EXPRESSION IN SUBQUERY %+v, %s, %d, %d",
								c.ObjectList, c.value, lastTermPos, inBracket), LogLevelInfo)
						}
					}
				} else {
					tokenList.Reset(tokenList.CurrentPos() - 1)
					break
				}
			} else {
				bracketTokenList = append(bracketTokenList, *t)
				bracketValue += (*t).Value()
				inBracket += 1
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("EXPRESSION IN '(' %+v, %s, %d, %d",
						bracketTokenList, bracketValue, lastTermPos, inBracket), LogLevelInfo)
				}
			}
		} else if (*t).Type() == "MySQLOperatorToken" && (*t).Value() == ")" {
			if inBracket == 0 {
				tokenList.Reset(tokenList.CurrentPos() - 1)
				break
			}
			bracketTokenList = append(bracketTokenList, *t)
			bracketValue += (*t).Value()
			inBracket -= 1
			if inBracket == 0 {
				for _, tmpt := range bracketTokenList {
					obj := tmpt.(MySQLObject)
					c.ObjectList = append(c.ObjectList, &obj)
				}
				bracketTokenList = make([]MySQLToken, 0)
				c.value += bracketValue
				bracketValue = ""
				lastTermPos = tokenList.CurrentPos()
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("EXPRESSION IN ')' %+v, %s, %d, %d",
						c.ObjectList, c.Value(), lastTermPos, inBracket), LogLevelInfo)
				}
			}
		} else if (*t).Type() == "MySQLKeywordToken" && InArray((*t).Value(), supportInFunctionKeyword) {
			if inBracket > 0 {
				bracketTokenList = append(bracketTokenList, *t)
				bracketValue += (*t).Value()
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("EXPRESSION IN ELSE %+v, %s, %d, %d",
						bracketTokenList, bracketValue, lastTermPos, inBracket), LogLevelInfo)
				}
			} else {
				tokenList.Reset(tokenList.CurrentPos() - 1)
				break
			}
		} else if (*t).Type() == "MySQLKeywordToken" && !InArray((*t).Value(), supportKeyword) &&
			!InArray((*t).Value(), supportFunction) && !InArray((*t).Value(), Keywords) {
			tokenList.Reset(tokenList.CurrentPos() - 1)
			break
		} else {
			if inBracket > 0 {
				bracketTokenList = append(bracketTokenList, *t)
				bracketValue += (*t).Value()
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("EXPRESSION IN ELSE %+v, %s, %d, %d",
						bracketTokenList, bracketValue, lastTermPos, inBracket), LogLevelInfo)
				}
			} else {
				obj := (*t).(MySQLObject)
				c.ObjectList = append(c.ObjectList, &obj)
				c.value += (*t).Value()
				if !InArray((*t).Type(), []string{"MySQLCommentToken", "MySQLSpaceToken"}) {
					lastTermPos = tokenList.CurrentPos()
				}
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("EXPRESSION IN ELSE %+v, %s, %d, %d",
						c.ObjectList, c.Value(), lastTermPos, inBracket), LogLevelInfo)
				}
			}
		}
	}
	if verboseFunc != nil {
		verboseFunc(fmt.Sprintf("EXPRESSION END %d, %d, %+v, %v",
			startPos, lastTermPos, bracketTokenList, tokenList.EOF()), LogLevelInfo)
	}
	if lastTermPos == startPos {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else if len(bracketTokenList) > 0 {
		tokenList.Reset(lastTermPos)
		return c, tokenList
	}
	return c, tokenList
}

// subpartitioning expr:
//     [LINEAR] HASH(expr)
//   | [LINEAR] KEY [ALGORITHM={1|2}] (column_list)

type MySQLSubPartitioningExpressionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLSubPartitioningExpressionComponent) Type() string {
	return "MySQLSubPartitioningExpressionComponent"
}

func (c *MySQLSubPartitioningExpressionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LINEAR",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "HASH",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALGORITHM",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "1",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "2",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLColumnNameListComponent",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLSubPartitioningExpressionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLSubPartitioningExpressionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// partitioning expr:
//     [LINEAR] HASH(expr)
//   | [LINEAR] KEY [ALGORITHM={1|2}] (column_list)
//   | RANGE{(expr) | COLUMNS(column_list)}
//   | LIST{(expr) | COLUMNS(column_list)}

type MySQLPartitioningExpressionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLPartitioningExpressionComponent) Type() string {
	return "MySQLPartitioningExpressionComponent"
}

func (c *MySQLPartitioningExpressionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LINEAR",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "HASH",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALGORITHM",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "1",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "2",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{5, 8},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLColumnNameListComponent",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RANGE",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LIST",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMNS",
			EndStatus:    8,
		},
	}
}

func NewMySQLPartitioningExpressionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLPartitioningExpressionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// assignment:
//    col_name = value

type MySQLAssignmentExpressionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLAssignmentExpressionComponent) Type() string {
	return "MySQLAssignmentExpressionComponent"
}

func (c *MySQLAssignmentExpressionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLAssignmentExpressionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLAssignmentExpressionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// assignment_list:
//    assignment [, assignment] ...

type MySQLAssignmentListExpressionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLAssignmentListExpressionComponent) Type() string {
	return "MySQLAssignmentListExpressionComponent"
}

func (c *MySQLAssignmentListExpressionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0, 2},
			AcceptObject: "MySQLAssignmentExpressionComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    2,
		},
	}
}

func NewMySQLAssignmentListExpressionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLAssignmentListExpressionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLIdentifierComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLIdentifierComponent) Type() string {
	return "MySQLIdentifierComponent"
}

func NewMySQLIdentifierComponent(tokenList MySQLTokenList,
	_ func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLIdentifierComponent{
		&MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	for !tokenList.EOF() {
		t := tokenList.Next()
		if t == nil {
			break
		}
		if (*t).Type() == "MySQLCommentToken" || (*t).Type() == "MySQLSpaceToken" {
			obj := (*t).(MySQLObject)
			c.ObjectList = append(c.ObjectList, &obj)
			c.value += (*t).Value()
			continue
		} else if (*t).Type() == "MySQLQuotedIdentifierToken" || (*t).Type() == "MySQLUnquotedIdentifierToken" {
			obj := (*t).(MySQLObject)
			c.ObjectList = append(c.ObjectList, &obj)
			c.value += (*t).Value()
			return c, tokenList
		} else if (*t).Type() == "MySQLKeywordToken" && InArray((*t).Value(), Keywords) {
			obj := (*t).(MySQLObject)
			c.ObjectList = append(c.ObjectList, &obj)
			c.value += (*t).Value()
			return c, tokenList
		} else {
			break
		}
	}
	tokenList.Reset(startPos)
	return nil, tokenList
}

type MySQLDatabaseNameComponent struct {
	*MySQLBaseComponent
	Database string
}

func (c *MySQLDatabaseNameComponent) Type() string {
	return "MySQLDatabaseNameComponent"
}

func (c *MySQLDatabaseNameComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLDatabaseNameComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLDatabaseNameComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLIdentifierComponent" {
				c.Database = strings.Trim((*t).Value(), "`")
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLTableNameComponent struct {
	*MySQLBaseComponent
	Database string
	Table    string
}

func (c *MySQLTableNameComponent) Type() string {
	return "MySQLTableNameComponent"
}

func (c *MySQLTableNameComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ".",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLTableNameComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLTableNameComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLIdentifierComponent" {
				if c.Table != "" {
					c.Database = c.Table
					c.Table = strings.Trim((*t).Value(), "`")
				} else {
					c.Table = strings.Trim((*t).Value(), "`")
				}
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLTableNameListComponent struct {
	*MySQLBaseComponent
	TableList []*MySQLTableNameComponent
}

func (c *MySQLTableNameListComponent) Type() string {
	return "MySQLTableNameListComponent"
}

func (c *MySQLTableNameListComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0, 2},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    2,
		},
	}
}

func NewMySQLTableNameListComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLTableNameListComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
		TableList: make([]*MySQLTableNameComponent, 0),
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLTableNameComponent" {
				c.TableList = append(c.TableList, (*t).(*MySQLTableNameComponent))
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLColumnNameComponent struct {
	*MySQLBaseComponent
	Database string
	Table    string
	Column   string
}

func (c *MySQLColumnNameComponent) Type() string {
	return "MySQLColumnNameComponent"
}

func (c *MySQLColumnNameComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ".",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ".",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLColumnNameComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLColumnNameComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1, 3}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLIdentifierComponent" {
				if c.Column != "" && c.Table != "" {
					c.Database = c.Table
					c.Table = c.Column
					c.Column = strings.Trim((*t).Value(), "`")
				} else if c.Column != "" {
					c.Table = c.Column
					c.Column = strings.Trim((*t).Value(), "`")
				} else {
					c.Column = strings.Trim((*t).Value(), "`")
				}
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLColumnNameListComponent struct {
	*MySQLBaseComponent
	ColumnList []*MySQLColumnNameComponent
}

func (c *MySQLColumnNameListComponent) Type() string {
	return "MySQLColumnNameListComponent"
}

func (c *MySQLColumnNameListComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0, 2},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    2,
		},
	}
}

func NewMySQLColumnNameListComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLColumnNameListComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
		ColumnList: make([]*MySQLColumnNameComponent, 0),
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLColumnNameComponent" {
				c.ColumnList = append(c.ColumnList, (*t).(*MySQLColumnNameComponent))
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLIndexColumnNameComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLIndexColumnNameComponent) Type() string {
	return "MySQLIndexColumnNameComponent"
}

func (c *MySQLIndexColumnNameComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{1, 4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ASC",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1, 4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESC",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLIndexColumnNameComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLIndexColumnNameComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1, 4}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLIndexColumnNameListComponent struct {
	*MySQLBaseComponent
	NameList []*MySQLIndexColumnNameComponent
}

func (c *MySQLIndexColumnNameListComponent) Type() string {
	return "MySQLIndexColumnNameListComponent"
}

func (c *MySQLIndexColumnNameListComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0, 2},
			AcceptObject: "MySQLIndexColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    2,
		},
	}
}

func NewMySQLIndexColumnNameListComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLIndexColumnNameListComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
		NameList: make([]*MySQLIndexColumnNameComponent, 0),
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLIndexColumnNameComponent" {
				c.NameList = append(c.NameList, (*t).(*MySQLIndexColumnNameComponent))
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLCharsetNameComponent struct {
	*MySQLBaseComponent
	Charset string
}

func (c *MySQLCharsetNameComponent) Type() string {
	return "MySQLCharsetNameComponent"
}

var supportCharsets = []string{
	"big5", "dec8", "cp850", "hp8", "koi8r",
	"latin1", "latin2", "swe7", "ascii", "ujis",
	"sjis", "hebrew", "tis620", "euckr", "koi8u",
	"gb2312", "greek", "cp1250", "gbk", "latin5",
	"armscii8", "utf8", "ucs2", "cp866", "keybcs2",
	"macce", "macroman", "cp852", "latin7", "utf8mb4",
	"cp1251", "utf16", "cp1256", "cp1257", "utf32",
	"binary", "geostd8", "cp932", "eucjpms",
}

func (c *MySQLCharsetNameComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLUnquotedIdentifierToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLCharsetNameComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLCharsetNameComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLUnquotedIdentifierToken" {
				c.Charset = strings.ToLower((*t).Value())
			}
		}
		if !InArray(c.Charset, supportCharsets) {
			tokenList.Reset(startPos)
			return nil, tokenList
		} else {
			tokenList.Reset(endPos)
			return c, tokenList
		}
	}
}

type MySQLCollationNameComponent struct {
	*MySQLBaseComponent
	Collation string
}

func (c *MySQLCollationNameComponent) Type() string {
	return "MySQLCollationNameComponent"
}

var supportSuffix = []string{
	"ai", "as", "ci", "cs", "bin",
}

func (c *MySQLCollationNameComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLUnquotedIdentifierToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLCollationNameComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLCollationNameComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "MySQLUnquotedIdentifierToken" {
				c.Collation = strings.ToLower((*t).Value())
			}
		}
		collationPiece := strings.Split(c.Collation, "_")
		if !InArray(collationPiece[0], supportCharsets) {
			tokenList.Reset(startPos)
			return nil, tokenList
		} else if !InArray(collationPiece[len(collationPiece)-1], supportSuffix) {
			tokenList.Reset(startPos)
			return nil, tokenList
		} else {
			tokenList.Reset(endPos)
			return c, tokenList
		}
	}
}

type MySQLEngineNameComponent struct {
	*MySQLBaseComponent
	Engine string
}

func (c *MySQLEngineNameComponent) Type() string {
	return "MySQLEngineNameComponent"
}

var supportEngine = []string{
	"INNODB", "MyISAM", "MEMORY", "CSV", "ARCHIVE",
	"BLACKHOLE", "MERGE", "FEDERATED",
}

func (c *MySQLEngineNameComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLUnquotedIdentifierToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLEngineNameComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLEngineNameComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if InArray((*t).Type(), []string{"MySQLUnquotedIdentifierToken", "MySQLKeywordToken"}) {
				c.Engine = strings.ToUpper((*t).Value())
			}
		}
		if !InArray(c.Engine, supportEngine) {
			tokenList.Reset(startPos)
			return nil, tokenList
		} else {
			tokenList.Reset(endPos)
			return c, tokenList
		}
	}
}

type MySQLNumericOptionValueComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLNumericOptionValueComponent) Type() string {
	return "MySQLNumericOptionValueComponent"
}

func (c *MySQLNumericOptionValueComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLNumericOptionValueComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLNumericOptionValueComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLBooleanOptionValueComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLBooleanOptionValueComponent) Type() string {
	return "MySQLBooleanOptionValueComponent"
}

func (c *MySQLBooleanOptionValueComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "0",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "1",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLBooleanOptionValueComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLBooleanOptionValueComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLStringOptionValueComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLStringOptionValueComponent) Type() string {
	return "MySQLStringOptionValueComponent"
}

func (c *MySQLStringOptionValueComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLStringOptionValueComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLStringOptionValueComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

//   [DEFAULT] CHARACTER SET [=] charset_name
// | [DEFAULT] COLLATE [=] collation_name

type MySQLDatabaseOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLDatabaseOptionComponent) Type() string {
	return "MySQLDatabaseOptionComponent"
}

func (c *MySQLDatabaseOptionComponent) GetFsmMap() []FsmMap {
	//   ／-------CHARACRTER--------↘              ／------charset_name-------↘
	// 0 --DEFAULT--> 1 --CHARACTER--> 2 --SET--> 3 -- = --> 4 --charset_name--->
	//                  ＼--COLLATE---> 5 -- = --> 6 -------collation_name-----↗
	//   ＼----------COLLATE--------↗   ＼-----------collation_name-----------↗
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARACTER",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLLATE",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{0, 1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARSET",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{3, 4},
			AcceptObject: "MySQLCharsetNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{5, 6},
			AcceptObject: "MySQLCollationNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLDatabaseOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLDatabaseOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// table_option:
//   AUTO_INCREMENT [=] value
// | AVG_ROW_LENGTH [=] value
// | [DEFAULT] CHARACTER SET [=] charset_name
// | CHECKSUM [=] {0 | 1}
// | [DEFAULT] COLLATE [=] collation_name
// | COMMENT [=] 'string'
// | CONNECTION [=] 'connect_string'
// | {DATA|INDEX} DIRECTORY [=] 'absolute path to directory'
// | DELAY_KEY_WRITE [=] {0 | 1}
// | ENGINE [=] engine_name
// | INSERT_METHOD [=] { NO | FIRST | LAST }
// | KEY_BLOCK_SIZE [=] value
// | MAX_ROWS [=] value
// | MIN_ROWS [=] value
// | PACK_KEYS [=] {0 | 1 | DEFAULT}
// | PASSWORD [=] 'string'
// | ROW_FORMAT [=] {DEFAULT|DYNAMIC|FIXED|COMPRESSED|REDUNDANT|COMPACT}
// | TABLESPACE tablespace_name [STORAGE {DISK|MEMORY|DEFAULT}]
// | UNION [=] (tbl_name[,tbl_name]...)

type MySQLTableOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLTableOptionComponent) Type() string {
	return "MySQLTableOptionComponent"
}

func (c *MySQLTableOptionComponent) GetFsmMap() []FsmMap {
	//0 --AUTO_INCREMENT--> 1 --numeric_option_value-->
	//  ＼--AVG_ROW_LENGTH--> 2 --numeric_option_value-↗
	//  ＼--database_option--↗
	//  ＼--CHECKSUM--> 3 --boolean_option_value-↗
	//  ＼--COMMENT--> 4 --string_option_value-↗
	//  ＼--CONNECTION--> 5 --string_option_value-↗
	//  ＼--DATA----> 6 --DIRECTORY--> 7 --string_option_value-↗
	//  ＼--INDEX--↗
	//  ＼--DELAY_KEY_WRITE--> 8 --boolean_option_value-↗
	//  ＼--ENGINE--> 9 -- = --> 25 --engine_name-↗
	//                  ＼------engine_name-------↗
	//  ＼--INSERT_METHOD--> 10 -- = --> 11 -----NO----↗
	//                                      ＼--FIRST--↗
	//                                      ＼---LAST--↗
	//                          ＼----------NO---------↗
	//                          ＼--------FIRST--------↗
	//                          ＼---------LAST--------↗
	//  ＼--KEY_BLOCK_SIZE--> 12 --numeric_option_value-->
	//  ＼--MAX_ROWS--> 13 --numeric_option_value-->
	//  ＼--MIN_ROWS--> 14 --numeric_option_value-->
	//  ＼--PACK_KEYS--> 15 -- = --> 16 -------0------↗
	//                                  ＼-----1------↗
	//                                  ＼---DEFAULT--↗
	//                      ＼------------0-----------↗
	//                      ＼------------1-----------↗
	//                      ＼---------DEFAULT--------↗
	//  ＼--PASSWORD--> 17 --string_option_value-↗
	//  ＼--ROW_FORMAT--> 18 -- = --> 19 -------DEFAULT------↗
	//                                  ＼------DYNAMIC------↗
	//                                  ＼--------FIXED------↗
	//                                  ＼-----COMPRESSED----↗
	//                                  ＼-----REDUNDANT-----↗
	//                                  ＼-------COMPACT-----↗
	//                       ＼-----------DEFAULT------------↗
	//                       ＼-----------DYNAMIC------------↗
	//                       ＼-------------FIXED------------↗
	//                       ＼----------COMPRESSED----------↗
	//                       ＼----------REDUNDANT-----------↗
	//                       ＼------------COMPACT-----------↗
	//  ＼--TABLESPACE--> 20 --tablespace_name--> 21 -->
	//                                               ＼--STORAGE--> 22 -----DISK----↗
	//                                                                 ＼---MEMORY--↗
	//                                                                 ＼--DEFAULT--↗
	//  ＼--UNION--> 23 --> = --> 24 --> table_name_list -->
	//                  ＼--------table_name_list---------↗
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AUTO_INCREMENT",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AVG_ROW_LENGTH",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLDatabaseOptionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHECKSUM",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLBooleanOptionValueComponent",
			AcceptValue:  "0",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COMMENT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CONNECTION",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DATA",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DIRECTORY",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DELAY_KEY_WRITE",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLBooleanOptionValueComponent",
			AcceptValue:  "0",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENGINE",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{9, 25},
			AcceptObject: "MySQLEngineNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INSERT_METHOD",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NO",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{10, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FIRST",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{10, 11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LAST",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY_BLOCK_SIZE",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MAX_ROWS",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MIN_ROWS",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PACK_KEYS",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{15, 16},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "0",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{15, 16},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "1",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{15, 16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PASSWORD",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLStringOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INSERT_METHOD",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DYNAMIC",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FIXED",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COMPRESSED",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REDUNDANT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{18, 19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COMPACT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLESPACE",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{20},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STORAGE",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISK",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MEMORY",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{22},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNION",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{23},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    24,
		},
		{
			StartStatus:  []int{23, 24},
			AcceptObject: "MySQLTableNameListComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLTableOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLTableOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{21}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// table_options:
//   table_option [[,] table_option] ...

type MySQLTableOptionListComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLTableOptionListComponent) Type() string {
	return "MySQLTableOptionListComponent"
}

func (c *MySQLTableOptionListComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0, 1, 2},
			AcceptObject: "MySQLTableOptionComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    2,
		},
	}
}

func NewMySQLTableOptionListComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLTableOptionListComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// index_type:
//   USING {BTREE | HASH}

type MySQLIndexTypeComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLIndexTypeComponent) Type() string {
	return "MySQLIndexTypeComponent"
}

func (c *MySQLIndexTypeComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "USING",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BTREE",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "HASH",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLIndexTypeComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLIndexTypeComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// index_option:
//   KEY_BLOCK_SIZE [=] value
// | index_type
// | WITH PARSER parser_name
// | COMMENT 'string'

type MySQLIndexOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLIndexOptionComponent) Type() string {
	return "MySQLIndexOptionComponent"
}

func (c *MySQLIndexOptionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY_BLOCK_SIZE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLNumericOptionValueComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLIndexTypeComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "WITH",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARSER",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COMMENT",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLIndexOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLIndexOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// reference_option:
//   RESTRICT | CASCADE | SET NULL | NO ACTION | SET DEFAULT

type MySQLReferenceOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLReferenceOptionComponent) Type() string {
	return "MySQLReferenceOptionComponent"
}

func (c *MySQLReferenceOptionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RESTRICT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CASCADE",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLNullToken",
			AcceptValue:  "NULL",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NO",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ACTION",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLReferenceOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLReferenceOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// partition_options:
//    PARTITION BY
//        { [LINEAR] HASH(expr)
//        | [LINEAR] KEY [ALGORITHM={1|2}] (column_list)
//        | RANGE{(expr) | COLUMNS(column_list)}
//        | LIST{(expr) | COLUMNS(column_list)} }
//    [PARTITIONS num]
//    [SUBPARTITION BY
//        { [LINEAR] HASH(expr)
//        | [LINEAR] KEY [ALGORITHM={1|2}] (column_list) }
//      [SUBPARTITIONS num]
//    ]
//    [(partition_definition [, partition_definition] ...)]

type MySQLPartitionOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLPartitionOptionComponent) Type() string {
	return "MySQLPartitionOptionComponent"
}

func (c *MySQLPartitionOptionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITION",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLPartitioningExpressionComponent",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITIONS",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{3, 5},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SUBPARTITION",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLSubPartitioningExpressionComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SUBPARTITIONS",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{3, 5, 8, 10, 12},
			AcceptObject: "MySQLPartitionDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    12,
		},
	}
}

func NewMySQLPartitionOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLPartitionOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{3, 5, 8, 10, 11}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// order_options:
//   {col_name | expr | position}
//   [ASC | DESC]

type MySQLOrderOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLOrderOptionComponent) Type() string {
	return "MySQLOrderOptionComponent"
}

func (c *MySQLOrderOptionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ASC",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DESC",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLOrderOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLOrderOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type MySQLOrderListOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLOrderListOptionComponent) Type() string {
	return "MySQLOrderListOptionComponent"
}

func (c *MySQLOrderListOptionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0, 2},
			AcceptObject: "MySQLOrderOptionComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    2,
		},
	}
}

func NewMySQLOrderListOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLOrderListOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// index_hint:
//    USE {INDEX|KEY}
//      [FOR {JOIN|ORDER BY|GROUP BY}] ([index_list])
//  | IGNORE {INDEX|KEY}
//      [FOR {JOIN|ORDER BY|GROUP BY}] (index_list)
//  | FORCE {INDEX|KEY}
//      [FOR {JOIN|ORDER BY|GROUP BY}] (index_list)

type MySQLIndexHintOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLIndexHintOptionComponent) Type() string {
	return "MySQLIndexHintOptionComponent"
}

func (c *MySQLIndexHintOptionComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "USE",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOR",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "JOIN",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ORDER",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "GROUP",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{2, 5},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6, 8},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{6, 7, 15},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IGNORE",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FORCE",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOR",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "JOIN",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ORDER",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "GROUP",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{10, 13},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14, 16},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    16,
		},
	}
}

func NewMySQLIndexHintOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLIndexHintOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// export_options:
//    [{FIELDS | COLUMNS}
//        [TERMINATED BY 'string']
//        [[OPTIONALLY] ENCLOSED BY 'char']
//        [ESCAPED BY 'char']
//    ]
//    [LINES
//        [STARTING BY 'string']
//        [TERMINATED BY 'string']
//    ]

type MySQLExportOptionComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLExportOptionComponent) Type() string {
	return "MySQLExportOptionComponent"
}

func (c *MySQLExportOptionComponent) GetFsmMap() []FsmMap {
	//TODO
	return []FsmMap{}
}

func NewMySQLExportOptionComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLExportOptionComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

type SubQueryComponent struct {
	*MySQLBaseComponent
	DatabaseList []string
	TableList    []string
}

func (c *SubQueryComponent) Type() string {
	return "SubQueryComponent"
}

func (c *SubQueryComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "UnionStatement",
			AcceptValue:  "",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "SelectStatement",
			AcceptValue:  "",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{2},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
	}
}

func NewSubQueryComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &SubQueryComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "SelectStatement" {
				c.DatabaseList = append(c.DatabaseList, (*t).(*SelectStatement).DatabaseList...)
				c.TableList = append(c.TableList, (*t).(*SelectStatement).TableList...)
			} else if (*t).Type() == "UnionStatement" {
				c.DatabaseList = append(c.DatabaseList, (*t).(*UnionStatement).DatabaseList...)
				c.TableList = append(c.TableList, (*t).(*UnionStatement).TableList...)
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// table_factor:
//    tbl_name [[AS] alias] [index_hint_list]
//  | table_subquery [AS] alias
//  | ( table_references )

type TableFactorComponent struct {
	*MySQLBaseComponent
	DatabaseList []string
	TableList    []string
}

func (c *TableFactorComponent) Type() string {
	return "TableFactorComponent"
}

func (c *TableFactorComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AS",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{1, 3, 5},
			AcceptObject: "MySQLIndexHintOptionComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "SubQueryComponent",
			AcceptValue:  "",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AS",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{6, 7},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "TableReferenceComponent",
			AcceptValue:  "",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
	}
}

func NewTableFactorComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &TableFactorComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1, 3, 4}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "SubQueryComponent" {
				c.DatabaseList = append(c.DatabaseList, (*t).(*SubQueryComponent).DatabaseList...)
				c.TableList = append(c.TableList, (*t).(*SubQueryComponent).TableList...)
			} else if (*t).Type() == "MySQLTableNameComponent" {
				c.DatabaseList = append(c.DatabaseList, (*t).(*MySQLTableNameComponent).Database)
				c.TableList = append(c.TableList, (*t).(*MySQLTableNameComponent).Table)
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// table_reference:
//    table_factor
//  | join_table
//
// join_table:
//    table_reference [INNER | CROSS] JOIN table_factor [join_condition]
//  | table_reference STRAIGHT_JOIN table_factor
//  | table_reference STRAIGHT_JOIN table_factor ON conditional_expr
//  | table_reference {LEFT|RIGHT} [OUTER] JOIN table_reference join_condition
//  | table_reference NATURAL [{LEFT|RIGHT} [OUTER]] JOIN table_factor
//
// join_condition:
//    ON conditional_expr
//  | USING (column_list)

type TableReferenceComponent struct {
	*MySQLBaseComponent
	DatabaseList []string
	TableList    []string
}

func (c *TableReferenceComponent) Type() string {
	return "TableReferenceComponent"
}

func (c *TableReferenceComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "TableFactorComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INNER",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CROSS",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "JOIN",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "TableFactorComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4, 10, 14},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ON",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLExpressionComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{4, 14},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "USING",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLColumnNameListComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "STRAIGHT_JOIN",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{9},
			AcceptObject: "TableFactorComponent",
			AcceptValue:  "",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LEFT",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RIGHT",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{11},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OUTER",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{11, 12},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "JOIN",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13},
			AcceptObject: "TableFactorComponent",
			AcceptValue:  "",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "NATURAL",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "LEFT",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{15},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RIGHT",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OUTER",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{15, 16, 17},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "JOIN",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{18},
			AcceptObject: "TableFactorComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
	}
}

func NewTableReferenceComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &TableReferenceComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1, 4, 10}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "TableFactorComponent" {
				c.DatabaseList = append(c.DatabaseList, (*t).(*TableFactorComponent).DatabaseList...)
				c.TableList = append(c.TableList, (*t).(*TableFactorComponent).TableList...)
			} else if (*t).Type() == "MySQLExpressionComponent" {
				for _, tmpT := range (*t).(*MySQLExpressionComponent).ObjectList {
					if (*tmpT).Type() == "SubQueryComponent" {
						c.DatabaseList = append(c.DatabaseList, (*tmpT).(*SubQueryComponent).DatabaseList...)
						c.TableList = append(c.TableList, (*tmpT).(*SubQueryComponent).TableList...)
					}
				}
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// table_references:
//    escaped_table_reference [, escaped_table_reference] ...
//
// escaped_table_reference:
//    table_reference
//  | { OJ table_reference }

type TableReferenceListComponent struct {
	*MySQLBaseComponent
	DatabaseList []string
	TableList    []string
}

func (c *TableReferenceListComponent) Type() string {
	return "TableReferenceListComponent"
}

func (c *TableReferenceListComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0, 2, 3},
			AcceptObject: "TableReferenceComponent",
			AcceptValue:  "",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{0, 3},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OJ",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    3,
		},
	}
}

func NewTableReferenceListComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &TableReferenceListComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
		DatabaseList: make([]string, 0),
		TableList:    make([]string, 0),
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{1}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		for _, t := range c.ObjectList {
			if (*t).Type() == "TableReferenceComponent" {
				c.DatabaseList = append(c.DatabaseList, (*t).(*TableReferenceComponent).DatabaseList...)
				c.TableList = append(c.TableList, (*t).(*TableReferenceComponent).TableList...)
			}
		}
		tokenList.Reset(endPos)
		return c, tokenList
	}
}

// alter_specification:
//   table_options
// | ADD [COLUMN] col_name column_definition
//       [FIRST | AFTER col_name]
// | ADD [COLUMN] (col_name column_definition,...)
// | ADD {INDEX|KEY} [index_name]
//       [index_type] (index_col_name,...) [index_option] ...
// | ADD [CONSTRAINT [symbol]] PRIMARY KEY
//       [index_type] (index_col_name,...) [index_option] ...
// | ADD [CONSTRAINT [symbol]]
//       UNIQUE [INDEX|KEY] [index_name]
//       [index_type] (index_col_name,...) [index_option] ...
// | ADD FULLTEXT [INDEX|KEY] [index_name]
//       (index_col_name,...) [index_option] ...
// | ADD SPATIAL [INDEX|KEY] [index_name]
//       (index_col_name,...) [index_option] ...
// | ADD [CONSTRAINT [symbol]]
//       FOREIGN KEY [index_name] (index_col_name,...)
//       reference_definition
// | ALTER [COLUMN] col_name {SET DEFAULT literal | DROP DEFAULT}
// | CHANGE [COLUMN] old_col_name new_col_name column_definition
//       [FIRST|AFTER col_name]
// | [DEFAULT] CHARACTER SET [=] charset_name [COLLATE [=] collation_name]
// | CONVERT TO CHARACTER SET charset_name [COLLATE collation_name]
// | {DISABLE|ENABLE} KEYS
// | {DISCARD|IMPORT} TABLESPACE
// | DROP [COLUMN] col_name
// | DROP {INDEX|KEY} index_name
// | DROP PRIMARY KEY
// | DROP FOREIGN KEY fk_symbol
// | FORCE
// | MODIFY [COLUMN] col_name column_definition
//       [FIRST | AFTER col_name]
// | ORDER BY col_name [, col_name] ...
// | RENAME [TO|AS] new_tbl_name
// | ADD PARTITION (partition_definition)
// | DROP PARTITION partition_names
// | TRUNCATE PARTITION {partition_names | ALL}
// | COALESCE PARTITION number
// | REORGANIZE PARTITION [partition_names INTO (partition_definitions)]
// | ANALYZE PARTITION {partition_names | ALL}
// | CHECK PARTITION {partition_names | ALL}
// | OPTIMIZE PARTITION {partition_names | ALL}
// | REBUILD PARTITION {partition_names | ALL}
// | REPAIR PARTITION {partition_names | ALL}
// | REMOVE PARTITIONING

type MySQLAlterTableSpecificationComponent struct {
	*MySQLBaseComponent
}

func (c *MySQLAlterTableSpecificationComponent) Type() string {
	return "MySQLAlterTableSpecificationComponent"
}

func (c *MySQLAlterTableSpecificationComponent) GetFsmMap() []FsmMap {
	return []FsmMap{
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLTableOptionListComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ADD",
			EndStatus:    1,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMN",
			EndStatus:    2,
		},
		{
			StartStatus:  []int{1, 2, 58, 59, 81},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    3,
		},
		{
			StartStatus:  []int{3},
			AcceptObject: "MySQLColumnDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    4,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FIRST",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{4},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AFTER",
			EndStatus:    5,
		},
		{
			StartStatus:  []int{5},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1, 2},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    6,
		},
		{
			StartStatus:  []int{6, 9},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    7,
		},
		{
			StartStatus:  []int{7},
			AcceptObject: "MySQLColumnDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    8,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    9,
		},
		{
			StartStatus:  []int{8},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1, 20},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{1, 20},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    10,
		},
		{
			StartStatus:  []int{10, 20},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{10, 11},
			AcceptObject: "MySQLIndexTypeComponent",
			AcceptValue:  "",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{10, 11, 12, 21, 22},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    13,
		},
		{
			StartStatus:  []int{13, 15},
			AcceptObject: "MySQLIndexColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    14,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{14},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    15,
		},
		{
			StartStatus:  []int{16},
			AcceptObject: "MySQLIndexOptionComponent",
			AcceptValue:  "",
			EndStatus:    16,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CONSTRAINT",
			EndStatus:    17,
		},
		{
			StartStatus:  []int{17},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    18,
		},
		{
			StartStatus:  []int{1, 17, 18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PRIMARY",
			EndStatus:    19,
		},
		{
			StartStatus:  []int{19},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    11,
		},
		{
			StartStatus:  []int{1, 17, 18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "UNIQUE",
			EndStatus:    20,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FULLTEXT",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SPATIAL",
			EndStatus:    21,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{21},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    22,
		},
		{
			StartStatus:  []int{21, 22},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    12,
		},
		{
			StartStatus:  []int{1, 17, 18},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOREIGN",
			EndStatus:    23,
		},
		{
			StartStatus:  []int{23},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    24,
		},
		{
			StartStatus:  []int{24},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    25,
		},
		{
			StartStatus:  []int{24, 25},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    26,
		},
		{
			StartStatus:  []int{26, 28},
			AcceptObject: "MySQLIndexColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    27,
		},
		{
			StartStatus:  []int{27},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    29,
		},
		{
			StartStatus:  []int{27},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    28,
		},
		{
			StartStatus:  []int{29},
			AcceptObject: "MySQLReferenceDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALTER",
			EndStatus:    30,
		},
		{
			StartStatus:  []int{30},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMN",
			EndStatus:    31,
		},
		{
			StartStatus:  []int{30, 31},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    32,
		},
		{
			StartStatus:  []int{32},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    33,
		},
		{
			StartStatus:  []int{33},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    34,
		},
		{
			StartStatus:  []int{34},
			AcceptObject: "MySQLStringToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{32},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DROP",
			EndStatus:    35,
		},
		{
			StartStatus:  []int{35},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHANGE",
			EndStatus:    36,
		},
		{
			StartStatus:  []int{36},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMN",
			EndStatus:    37,
		},
		{
			StartStatus:  []int{36, 37},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    81,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DEFAULT",
			EndStatus:    38,
		},
		{
			StartStatus:  []int{0, 38},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARACTER",
			EndStatus:    39,
		},
		{
			StartStatus:  []int{39},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    40,
		},
		{
			StartStatus:  []int{0, 38},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARSET",
			EndStatus:    40,
		},
		{
			StartStatus:  []int{40},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    41,
		},
		{
			StartStatus:  []int{40, 41},
			AcceptObject: "MySQLCharsetNameComponent",
			AcceptValue:  "",
			EndStatus:    42,
		},
		{
			StartStatus:  []int{42},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLLATE",
			EndStatus:    43,
		},
		{
			StartStatus:  []int{43},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "=",
			EndStatus:    44,
		},
		{
			StartStatus:  []int{43, 44},
			AcceptObject: "MySQLCollationNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CONVERT",
			EndStatus:    45,
		},
		{
			StartStatus:  []int{45},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TO",
			EndStatus:    46,
		},
		{
			StartStatus:  []int{46},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARACTER",
			EndStatus:    47,
		},
		{
			StartStatus:  []int{47},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "SET",
			EndStatus:    48,
		},
		{
			StartStatus:  []int{46},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHARSET",
			EndStatus:    48,
		},
		{
			StartStatus:  []int{48},
			AcceptObject: "MySQLCharsetNameComponent",
			AcceptValue:  "",
			EndStatus:    49,
		},
		{
			StartStatus:  []int{49},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLLATE",
			EndStatus:    50,
		},
		{
			StartStatus:  []int{50},
			AcceptObject: "MySQLCollationNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISABLE",
			EndStatus:    51,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ENABLE",
			EndStatus:    51,
		},
		{
			StartStatus:  []int{51},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEYS",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DISCARD",
			EndStatus:    52,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "IMPORT",
			EndStatus:    52,
		},
		{
			StartStatus:  []int{52},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TABLESPACE",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "DROP",
			EndStatus:    53,
		},
		{
			StartStatus:  []int{53},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMN",
			EndStatus:    54,
		},
		{
			StartStatus:  []int{53, 54},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{53},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INDEX",
			EndStatus:    55,
		},
		{
			StartStatus:  []int{53},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    55,
		},
		{
			StartStatus:  []int{55},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{53},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PRIMARY",
			EndStatus:    56,
		},
		{
			StartStatus:  []int{56},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{53},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FOREIGN",
			EndStatus:    57,
		},
		{
			StartStatus:  []int{57},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "KEY",
			EndStatus:    55,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "FORCE",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "MODIFY",
			EndStatus:    58,
		},
		{
			StartStatus:  []int{58},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COLUMN",
			EndStatus:    59,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ORDER",
			EndStatus:    60,
		},
		{
			StartStatus:  []int{60},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "BY",
			EndStatus:    61,
		},
		{
			StartStatus:  []int{61},
			AcceptObject: "MySQLColumnNameComponent",
			AcceptValue:  "",
			EndStatus:    62,
		},
		{
			StartStatus:  []int{62},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    61,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "RENAME",
			EndStatus:    63,
		},
		{
			StartStatus:  []int{63},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TO",
			EndStatus:    64,
		},
		{
			StartStatus:  []int{63},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "AS",
			EndStatus:    64,
		},
		{
			StartStatus:  []int{63, 64},
			AcceptObject: "MySQLTableNameComponent",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{1},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITION",
			EndStatus:    65,
		},
		{
			StartStatus:  []int{65},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  "(",
			EndStatus:    66,
		},
		{
			StartStatus:  []int{66},
			AcceptObject: "MySQLPartitionDefinitionComponent",
			AcceptValue:  "",
			EndStatus:    67,
		},
		{
			StartStatus:  []int{67},
			AcceptObject: "MySQLOperatorToken",
			AcceptValue:  ")",
			EndStatus:    68,
		},
		{
			StartStatus:  []int{53},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITION",
			EndStatus:    69,
		},
		{
			StartStatus:  []int{69, 72},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    70,
		},
		{
			StartStatus:  []int{70},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    69,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "TRUNCATE",
			EndStatus:    71,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ANALYZE",
			EndStatus:    71,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "CHECK",
			EndStatus:    71,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "OPTIMIZE",
			EndStatus:    71,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REBUILD",
			EndStatus:    71,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REPAIR",
			EndStatus:    71,
		},
		{
			StartStatus:  []int{71},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITION",
			EndStatus:    72,
		},
		{
			StartStatus:  []int{72},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "ALL",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "COALESCE",
			EndStatus:    73,
		},
		{
			StartStatus:  []int{73},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITION",
			EndStatus:    74,
		},
		{
			StartStatus:  []int{74},
			AcceptObject: "MySQLNumericToken",
			AcceptValue:  "",
			EndStatus:    FinalStatus,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REORGANIZE",
			EndStatus:    75,
		},
		{
			StartStatus:  []int{75},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITION",
			EndStatus:    76,
		},
		{
			StartStatus:  []int{76},
			AcceptObject: "MySQLIdentifierComponent",
			AcceptValue:  "",
			EndStatus:    77,
		},
		{
			StartStatus:  []int{77},
			AcceptObject: "MySQLDelimiterToken",
			AcceptValue:  ",",
			EndStatus:    76,
		},
		{
			StartStatus:  []int{77},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "INTO",
			EndStatus:    65,
		},
		{
			StartStatus:  []int{0},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "REMOVE",
			EndStatus:    78,
		},
		{
			StartStatus:  []int{78},
			AcceptObject: "MySQLKeywordToken",
			AcceptValue:  "PARTITIONING",
			EndStatus:    FinalStatus,
		},
	}
}

func NewMySQLAlterTableSpecificationComponent(tokenList MySQLTokenList,
	verboseFunc func(message string, level LogLevel)) (MySQLComponent, MySQLTokenList) {
	startPos := tokenList.CurrentPos()
	c := &MySQLAlterTableSpecificationComponent{
		MySQLBaseComponent: &MySQLBaseComponent{
			ObjectList: make([]*MySQLObject, 0),
		},
	}
	endPos := c.ParseByFsm(c.GetFsmMap(), tokenList, []int{4, 16, 42, 49, 62, 68, 70, 76}, verboseFunc)
	if endPos == -1 {
		tokenList.Reset(startPos)
		return nil, tokenList
	} else {
		tokenList.Reset(endPos)
		return c, tokenList
	}
}
