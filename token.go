package mysqlparser_go

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type MySQLToken interface {
	Value() string
	Type() string
}

type MySQLTokenList struct {
	tokenList []*MySQLToken
	curIndex  int
}

var parseList = []func(sql string) (MySQLToken, error, string){
	NewMySQLDelimiterToken,
	NewMySQLNullToken,
	NewMySQLSpaceToken,
	NewMySQLCommentToken,
	NewMySQLStringToken,
	NewMySQLQuotedIdentifierToken,
	NewMySQLOperatorToken,
	NewMySQLNumericToken,
	NewMySQLHexadecimalToken,
	NewMySQLBitToken,
	NewMySQLVariableToken,
	NewMySQLKeywordToken,
	NewMySQLUnquotedIdentifierToken,
}

// NewMySQLTokenList ...
func NewMySQLTokenList(sql string, verboseFunc func(message string, level LogLevel)) (
	MySQLTokenList, error) {
	list := MySQLTokenList{}
	list.tokenList = make([]*MySQLToken, 0)
	for len(sql) > 0 {
		parsed := false
		for _, parser := range parseList {
			var token MySQLToken
			var err error
			token, err, sql = parser(sql)
			if err != nil {
				return list, err
			}
			if token != nil {
				list.tokenList = append(list.tokenList, &token)
				if verboseFunc != nil {
					verboseFunc(fmt.Sprintf("PARSED TOKEN %s: %s", token.Type(), token.Value()),
						LogLevelInfo)
				}
				parsed = true
				break
			}
		}
		if !parsed {
			return list, errors.New(fmt.Sprintf("parse error on %s", sql))
		}
	}
	return list, nil
}

// ToString ...
func (l *MySQLTokenList) ToString() string {
	strList := make([]string, len(l.tokenList))
	for index, token := range l.tokenList {
		strList[index] = fmt.Sprintf("<%s: %s>", (*token).Type(), (*token).Value())
	}
	return strings.Join(strList, "\n")
}

// Next ...
func (l *MySQLTokenList) Next() *MySQLToken {
	l.curIndex += 1
	if l.curIndex > len(l.tokenList) {
		return nil
	}
	return l.tokenList[l.curIndex-1]
}

// Reset ...
func (l *MySQLTokenList) Reset(pos int) {
	l.curIndex = pos
}

// CurrentPos ...
func (l *MySQLTokenList) CurrentPos() int {
	return l.curIndex
}

// EOF ...
func (l *MySQLTokenList) EOF() bool {
	if l.curIndex > len(l.tokenList) {
		return true
	}
	return false
}

// Divide 拆分多句SQL
func (l *MySQLTokenList) Divide() []MySQLTokenList {
	tokenListList := make([]MySQLTokenList, 0)
	l.Reset(0)
	status, start, end := 0, 0, 0
	currentTokenList, err := NewMySQLTokenList("", nil)
	if err != nil {
		return tokenListList
	}
	for !l.EOF() {
		token := l.Next()
		if token == nil {
			break
		}
		if (*token).Type() == "MySQLDelimiterToken" && (*token).Value() == ";" {
			if start != end {
				currentTokenList.tokenList = l.tokenList[start:end]
				tokenListList = append(tokenListList, currentTokenList)
				currentTokenList, _ = NewMySQLTokenList("", nil)
				start = end
			}
			status = 0
		} else if status == 0 {
			start = l.CurrentPos()
			end = l.CurrentPos()
			if (*token).Type() != "MySQLSpaceToken" {
				start = l.CurrentPos() - 1
				status = 1
			}
		} else {
			if (*token).Type() != "MySQLSpaceToken" {
				end = l.CurrentPos()
			}
		}
	}
	if start != end {
		currentTokenList.tokenList = l.tokenList[start:end]
		tokenListList = append(tokenListList, currentTokenList)
	}
	return tokenListList
}

// HasToken 判断是否存在token
func (l *MySQLTokenList) HasToken(tokenType string, tokenValue string) bool {
	for _, token := range l.tokenList {
		if (*token).Type() == tokenType && (*token).Value() == tokenValue {
			return true
		}
	}
	return false
}

// GetNextValidToken 取num个后续token
func (l *MySQLTokenList) GetNextValidToken(num int) []*MySQLToken {
	returnVal := make([]*MySQLToken, 0)
	for _, token := range l.tokenList[l.curIndex:] {
		if (*token).Type() == "MySQLSpaceToken" || (*token).Type() == "MySQLCommentToken" {
			continue
		} else {
			returnVal = append(returnVal, token)
		}
		if len(returnVal) == num {
			return returnVal
		}
	}
	return returnVal
}

type MySQLBitToken struct {
	value string
}

func NewMySQLBitToken(sql string) (MySQLToken, error, string) {
	if sql[0] != '0' && sql[0] != 'B' && sql[0] != 'b' {
		return nil, nil, sql
	}
	bRegex, err := regexp.Compile("(?i)^B'[01]+'")
	if err != nil {
		return nil, err, sql
	}
	dRegex, err := regexp.Compile("^0b[01]+")
	if err != nil {
		return nil, err, sql
	}

	bTest := bRegex.MatchString(sql)
	dTest := dRegex.MatchString(sql)
	token := MySQLBitToken{}
	if bTest {
		token.value = bRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if dTest {
		token.value = dRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLBitToken) Value() string {
	return t.value
}

func (t *MySQLBitToken) Type() string {
	return "MySQLBitToken"
}

type MySQLCommentToken struct {
	value string
}

func NewMySQLCommentToken(sql string) (MySQLToken, error, string) {
	if sql[0] != '-' && sql[0] != '/' && sql[0] != '#' {
		return nil, nil, sql
	}
	singleLineRegex, err := regexp.Compile("^(--\\s+|#).*?(\\r\\n|\\r|\\n|$)")
	if err != nil {
		return nil, err, sql
	}
	multiLineRegex, err := regexp.Compile("(?m)^/\\*.*?\\*/")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLCommentToken{}
	singleTest := singleLineRegex.MatchString(sql)
	multiTest := multiLineRegex.MatchString(sql)
	if singleTest {
		token.value = singleLineRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if multiTest {
		token.value = multiLineRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLCommentToken) Value() string {
	return t.value
}

func (t *MySQLCommentToken) Type() string {
	return "MySQLCommentToken"
}

type MySQLDelimiterToken struct {
	value string
}

func NewMySQLDelimiterToken(sql string) (MySQLToken, error, string) {
	if sql[0] != ',' && sql[0] != ';' {
		return nil, nil, sql
	}
	token := MySQLDelimiterToken{}
	token.value = sql[:1]
	sql = sql[1:]
	return &token, nil, sql
}

func (t *MySQLDelimiterToken) Value() string {
	return t.value
}

func (t *MySQLDelimiterToken) Type() string {
	return "MySQLDelimiterToken"
}

type MySQLHexadecimalToken struct {
	value string
}

func NewMySQLHexadecimalToken(sql string) (MySQLToken, error, string) {
	if sql[0] != '0' && sql[0] != 'X' && sql[0] != 'x' {
		return nil, nil, sql
	}
	xRegex, err := regexp.Compile("(?i)^X'([0-9A-F][0-9A-F])+'")
	if err != nil {
		return nil, err, sql
	}
	dRegex, err := regexp.Compile("^0x([0-9A-Fa-f][0-9A-Fa-f])+")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLHexadecimalToken{}
	xTest := xRegex.MatchString(sql)
	dTest := dRegex.MatchString(sql)
	if xTest {
		token.value = xRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if dTest {
		token.value = dRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLHexadecimalToken) Value() string {
	return t.value
}

func (t *MySQLHexadecimalToken) Type() string {
	return "MySQLHexadecimalToken"
}

type MySQLKeywordToken struct {
	value string
}

var (
	Keywords = []string{
		"ABS", "ACOS", "ACTION", "ADDDATE", "ADDTIME",
		"AES_DECRYPT", "AES_ENCRYPT", "AFTER", "AGAINST", "AGGREGATE",
		"ALGORITHM", "ANY", "ASCII", "ASIN", "AT",
		"ATAN", "ATAN2", "AUTHORS", "AUTO_INCREMENT", "AUTOEXTEND_SIZE",
		"AVG", "AVG_ROW_LENGTH", "BACKUP", "BEGIN", "BENCHMARK",
		"BIN", "BINLOG", "BIT", "BIT_AND", "BIT_COUNT",
		"BIT_LENGTH", "BIT_OR", "BIT_XOR", "BLOCK", "BOOL",
		"BOOLEAN", "BTREE", "BYTE", "CACHE", "CASCADED",
		"CAST", "CATALOG_NAME", "CCONCAT_WS", "CEIL", "CEILING",
		"CHAIN", "CHANGED", "CHAR_LENGTH", "CHARACTER_LENGTH", "CHARSET",
		"CHECKSUM", "CIPHER", "CLASS_ORIGIN", "CLIENT", "CLOSE",
		"COALESCE", "CODE", "COERCIBILITY", "COLLATION", "COLUMN_NAME",
		"COLUMNS", "COMMENT", "COMMIT", "COMMITTED", "COMPACT",
		"COMPLETION", "COMPRESS", "COMPRESSED", "CONCAT", "CONCURRENT",
		"CONNECTION", "CONNECTION_ID", "CONSISTENT", "CONSTRAINT_CATALOG", "CONSTRAINT_NAME",
		"CONSTRAINT_SCHEMA", "CONTAINS", "CONTEXT", "CONTRIBUTORS", "CONV",
		"CONVERT_TZ", "COS", "COT", "COUNT", "CPU",
		"CRC32", "CUBE", "CURDATE", "CURSOR_NAME", "CURTIME",
		"DATA", "DATAFILE", "DATE", "DATE_ADD", "DATE_FORMAT",
		"DATE_SUB", "DATEDIFF", "DATETIME", "DAY", "DAYNAME",
		"DAYOFMONTH", "DAYOFWEEK", "DAYOFYEAR", "DEALLOCATE", "DECODE",
		"DEFINER", "DEGREES", "DELAY_KEY_WRITE", "DES_DECRYPT", "DES_ENCRYPT",
		"DES_KEY_FILE", "DIRECTORY", "DISABLE", "DISCARD", "DISK",
		"DO", "DUMPFILE", "DUPLICATE", "DYNAMIC", "ELT",
		"ENABLE", "ENCODE", "ENCRYPT", "END", "ENDS",
		"ENGINE", "ENGINES", "ENUM", "ERROR", "ERRORS",
		"ESCAPE", "EVENT", "EVENTS", "EVERY", "EXECUTE",
		"EXP", "EXPANSION", "EXPORT_SET", "EXTENDED", "EXTENT_SIZE",
		"EXTRACT", "FAST", "FAULTS", "FIELD", "FIELDS",
		"FILE", "FIND_IN_SET", "FIRST", "FIXED", "FLOOR",
		"FLUSH", "FORM_UNIXTIME", "FORMAT", "FOUND", "FOUND_ROWS",
		"FRAC_SECOND", "FROM_DAYS", "FULL", "FUNCTION", "GEOMETRY",
		"GEOMETRYCOLLECTION", "GET_FORMAT", "GET_LOCK", "GLOBAL", "GRANTS",
		"GROUP_CONCAT", "HANDLER", "HASH", "HELP", "HEX",
		"HOST", "HOSTS", "HOUR", "IDENTIFIED", "IFNULL",
		"IGNORE_SERVER_IDS", "IMPORT", "INDEXES", "INET_ATON", "INET_NTOA",
		"INITIAL_SIZE", "INNOBASE", "INNODB", "INSERT_METHOD", "INSTALL",
		"INSTR", "INTERNAL", "INTO", "INVOKER", "IO", "IO_THREAD",
		"IPC", "IS_FREE_LOCK", "IS_USED_LOCK", "ISOLATION", "ISSUER",
		"KEY_BLOCK_SIZE", "LANGUAGE", "LAST", "LAST_DAY", "LAST_INSERT_ID",
		"LCASE", "LEAVES", "LENGTH", "LESS", "LEVEL",
		"LINESTRING", "LIST", "LN", "LOAD_FILE", "LOCAL",
		"LOCATE", "LOCKS", "LOG", "LOG10", "LOG2",
		"LOGFILE", "LOGS", "LOWER", "LPAD", "LTRIM",
		"MAKE_SET", "MAKEDATE", "MAKETIME", "MASTER", "MASTER_CONNECT_RETRY",
		"MASTER_HEARTBEAT_PERIOD", "MASTER_HOST", "MASTER_LOG_FILE", "MASTER_LOG_POS", "MASTER_PASSWORD",
		"MASTER_PORT", "MASTER_POS_WAIT", "MASTER_SERVER_ID", "MASTER_SSL", "MASTER_SSL_CA",
		"MASTER_SSL_CAPATH", "MASTER_SSL_CERT", "MASTER_SSL_CIPHER", "MASTER_SSL_KEY", "MASTER_USER",
		"MAX", "MAX_CONNECTIONS_PER_HOUR", "MAX_QUERIES_PER_HOUR", "MAX_ROWS", "MAX_SIZE",
		"MAX_UPDATES_PER_HOUR", "MAX_USER_CONNECTIONS", "MD5", "MEDIUM", "MEMORY",
		"MERGE", "MESSAGE_TEXT", "MICROSECOND", "MID", "MIGRATE",
		"MIN", "MIN_ROWS", "MINUTE", "MODE", "MODIFY",
		"MONTH", "MONTHNAME", "MULTILINESTRING", "MULTIPOINT", "MULTIPOLYGON",
		"MUTEX", "MYSQL_ERRNO", "NAME", "NAME_CONST", "NAMES",
		"NATIONAL", "NCHAR", "NDB", "NDBCLUSTER", "NEW",
		"NEXT", "NO", "NO_WAIT", "NODEGROUP", "NONE",
		"NOW", "NULLIF", "NVARCHAR", "OCT", "OCTET_LENGTH",
		"OFFSET", "OJ", "OLD_PASSWORD", "ONE", "ONE_SHOT",
		"OPEN", "OPTIONS", "ORD", "OWNER", "PACK_KEYS",
		"PAGE", "PARSER", "PARTIAL", "PARTITION", "PARTITIONING",
		"PARTITIONS", "PASSWORD", "PERIOD_ADD", "PERIOD_DIFF", "PHASE",
		"PI", "PLUGIN", "PLUGINS", "POINT", "POLYGON", "PORT",
		"POSITION", "POW", "POWER", "PREPARE", "PRESERVE",
		"PREV", "PRIVILEGES", "PROCESSLIST", "PROFILE", "PROFILES",
		"PROXY", "QUARTER", "QUERY", "QUICK", "QUOTE",
		"RADIANS", "RAND", "READ_ONLY", "REBUILD", "RECOVER",
		"REDO_BUFFER_SIZE", "REDOFILE", "REDUNDANT", "RELAY", "RELAY_LOG_FILE",
		"RELAY_LOG_POS", "RELAY_THREAD", "RELAYLOG", "RELEASE_LOCK", "RELOAD",
		"REMOVE", "REORGANIZE", "REPAIR", "REPEATABLE", "REPLICATION",
		"RESET", "RESTORE", "RESUME", "RETURNS", "REVERSE",
		"ROLLBACK", "ROLLUP", "ROUND", "ROUTINE", "ROW",
		"ROW_COUNT", "ROW_FORMAT", "ROWS", "RPAD", "RTREE",
		"RTRIM", "SAVEPOINT", "SCHEDULE", "SCHEMA_NAME", "SECOND",
		"SECURITY", "SERIAL", "SERIALIZABLE", "SERVER", "SESSION",
		"SESSION_USER", "SET_TO_TIME", "SHA", "SHA1", "SHA2",
		"SHARE", "SHUTDOWN", "SIGN", "SIGNED", "SIMPLE",
		"SIN", "SLAVE", "SLEEP", "SNAPSHOT", "SOCKET",
		"SOME", "SONAME", "SOUNDEX", "SOUNDS", "SOURCE",
		"SPACE", "SQL_BUFFER_RESULT", "SQL_CACHE", "SQL_NO_CACHE", "SQL_THREAD",
		"SQL_TSI_DAY", "SQL_TSI_FRAC_SECOND", "SQL_TSI_HOUR", "SQL_TSI_MINUTE", "SQL_TSI_MONTH",
		"SQL_TSI_QUARTER", "SQL_TSI_SECOND", "SQL_TSI_WEEK", "SQL_TSI_YEAR", "SQRT",
		"START", "STARTS", "STATUS", "STD", "STDDEV",
		"STDDEV_POP", "STDDEV_SAMP", "STOP", "STORAGE", "STR_TO_DATE",
		"STRCMP", "STRING", "SUBCLASS_ORIGIN", "SUBDATE", "SUBJECT",
		"SUBPARTITION", "SUBPARTITIONS", "SUBSTR", "SUBSTRING", "SUBSTRING_INDEX",
		"SUM", "SUPER", "SUSPEND", "SWAPS", "SWITCHES",
		"SYSDATE", "SYSTEM_USER", "TABLE_CHECKSUM", "TABLE_NAME", "TABLES",
		"TABLESPACE", "TAN", "TEMPORARY", "TEMPTABLE", "TEXT",
		"THAN", "TIME", "TIME_FORMAT", "TIME_TO_SEC", "TIMEDIFF",
		"TIMESTAMP", "TIMESTAMPADD", "TIMESTAMPDIFF", "TO_DAYS", "TO_SECONDS",
		"TRANSACTION", "TRIGGERS", "TRIM", "TRUNCATE", "TYPE",
		"TYPES", "UCASE", "UNCOMMITTED", "UNCOMPRESS", "UNCOMPRESSED_LENGTH",
		"UNDEFINED", "UNDO_BUFFER_SIZE", "UNDOFILE", "UNHEX", "UNICODE",
		"UNINSTALL", "UNIX_TIMESTAMP", "UNKNOWN", "UNTIL", "UPGRADE",
		"UPPER", "USE_FRM", "USER", "USER_RESOURCES", "UUID",
		"UUID_SHORT", "VALUE", "VAR_POP", "VAR_SAMP", "VARIABLES",
		"VARIANCE", "VERSION", "VIEW", "WAIT", "WARNINGS",
		"WEEK", "WEEKDAY", "WEEKOFYEAR", "WORK", "WRAPPER",
		"X509", "XA", "XML", "YEAR", "YEARWEEK",
	}
	reservedKeywords = []string{
		"ACCESSIBLE", "ADD", "ALL", "ALTER", "ANALYZE",
		"AND", "AS", "ASC", "ASENSITIVE", "BEFORE",
		"BETWEEN", "BIGINT", "BINARY", "BLOB", "BOTH",
		"BY", "CALL", "CASCADE", "CASE", "CHANGE",
		"CHAR", "CHARACTER", "CHECK", "COLLATE", "COLUMN",
		"CONDITION", "CONSTRAINT", "CONTINUE", "CONVERT", "CREATE",
		"CROSS", "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP", "CURRENT_USER",
		"CURSOR", "DATABASE", "DATABASES", "DAY_HOUR", "DAY_MICROSECOND",
		"DAY_MINUTE", "DAY_SECOND", "DEC", "DECIMAL", "DECLARE",
		"DEFAULT", "DELAYED", "DELETE", "DESC", "DESCRIBE",
		"DETERMINISTIC", "DISTINCT", "DISTINCTROW", "DIV", "DOUBLE",
		"DROP", "DUAL", "EACH", "ELSE", "ELSEIF",
		"ENCLOSED", "ESCAPED", "EXISTS", "EXIT", "EXPLAIN",
		"FALSE", "FETCH", "FLOAT", "FLOAT4", "FLOAT8",
		"FOR", "FORCE", "FOREIGN", "FROM", "FULLTEXT",
		"GENERAL", "GRANT", "GROUP", "HAVING", "HIGH_PRIORITY",
		"HOUR_MICROSECOND", "HOUR_MINUTE", "HOUR_SECOND", "IF", "IGNORE",
		"IN", "INDEX", "INFILE", "INNER", "INOUT",
		"INSENSITIVE", "INSERT", "INT", "INT1", "INT2",
		"INT3", "INT4", "INT8", "INTERGER", "INTERVAL",
		"INFO", "IS", "ITERATE", "JOIN", "KEY",
		"KEYS", "KILL", "LEADING", "LEAVE", "LEFT",
		"LIKE", "LIMIT", "LINEAR", "LINES", "LOAD",
		"LOCALTIME", "LOCALTIMESTAMP", "LOCK", "LONG", "LONGBLOB",
		"LONGTEXT", "LOOP", "LOW_PRIORITY", "MASTER_SSL_VERIFY_SERVER_CERT", "MATCH",
		"MAXVALUE", "MEDIUMBLOB", "MEDIUMINT", "MEDIUMTEXT", "MIDDLEINT",
		"MINUTE_MICROSECOND", "MINUTE_SECOND", "MOD", "MODIFIES", "NATURAL",
		"NOT", "NO_WRITE_TO_BINLOG", "NULL", "NUMERIC", "ON",
		"OPTIMIZE", "OPTION", "OPTIONALLY", "OR", "ORDER",
		"OUT", "OUTER", "OUTFILE", "PRECISION", "PRIMARY",
		"PROCEDURE", "PURGE", "RANGE", "READ", "READS",
		"READ_WRITE", "REAL", "REFERENCES", "REGEXP", "RELEASE",
		"RENAME", "REPEAT", "REPLACE", "REQUIRE", "RESIGNAL",
		"RESTRICT", "RETURN", "REVOKE", "RIGHT", "RLIKE",
		"SCHEMA", "SCHEMAS", "SECOND_MICROSECOND", "SELECT", "SENSITIVE",
		"SEPARATOR", "SET", "SHOW", "SIGNAL", "SLOW",
		"SMALLINT", "SPATIAL", "SPECIFIC", "SQL", "SQLEXCEPTION",
		"SQLSTATE", "SQLWARNING", "SQL_BIG_RESULT", "SQL_CALC_FOUND_ROWS", "SQL_SMALL_RESULT",
		"SSL", "STARTING", "STRAIGHT_JOIN", "TABLE", "TERMINATED",
		"THEN", "TINYBLOB", "TINYINT", "TINYTEXT", "TO",
		"TRAILING", "TRIGGER", "TRUE", "UNDO", "UNION",
		"UNIQUE", "UNLOCK", "UNSIGNED", "UPDATE", "USAGE",
		"USE", "USING", "UTC_DATE", "UTC_TIME", "UTC_TIMESTAMP",
		"VALUES", "VARBINARY", "VARCHAR", "VARCHARACTER", "VARYING",
		"WHEN", "WHERE", "WHILE", "WITH", "WRITE",
		"XOR", "YEAR_MONTH", "ZEROFILL",
	}
)

func NewMySQLKeywordToken(sql string) (MySQLToken, error, string) {
	startChar := strings.ToLower(sql)[0]
	if startChar < 'a' || startChar > 'z' {
		return nil, nil, sql
	}
	reservedKeywordsRegex, err := regexp.Compile("(?i)^\\b(" + strings.Join(reservedKeywords, "|") + ")\\b")
	if err != nil {
		return nil, err, sql
	}
	token := MySQLKeywordToken{}
	reservedKeywordsTest := reservedKeywordsRegex.MatchString(sql)
	if reservedKeywordsTest {
		token.value = strings.ToUpper(reservedKeywordsRegex.FindStringSubmatch(sql)[0])
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	keywordsRegex, err := regexp.Compile("(?i)^\\b(" + strings.Join(Keywords, "|") + ")\\b")
	if err != nil {
		return nil, err, sql
	}
	keywordsTest := keywordsRegex.MatchString(sql)
	if keywordsTest {
		token.value = strings.ToUpper(keywordsRegex.FindStringSubmatch(sql)[0])
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func NewMySQLKeywordTokenByExpectKeywords(sql string, expectKeywords []string) (MySQLToken, error, string) {
	startChar := strings.ToLower(sql)[0]
	if startChar < 'a' || startChar > 'z' {
		return nil, nil, sql
	}
	expectKeywordsRegex, err := regexp.Compile("(?i)^\\b(" + strings.Join(expectKeywords, "|") + ")\\b")
	if err != nil {
		return nil, err, sql
	}
	token := MySQLKeywordToken{}
	expectKeywordsTest := expectKeywordsRegex.MatchString(sql)
	if expectKeywordsTest {
		token.value = strings.ToUpper(expectKeywordsRegex.FindStringSubmatch(sql)[0])
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLKeywordToken) Value() string {
	return t.value
}

func (t *MySQLKeywordToken) Type() string {
	return "MySQLKeywordToken"
}

type MySQLNullToken struct {
	value string
}

func NewMySQLNullToken(sql string) (MySQLToken, error, string) {
	if sql[0] != 'N' && sql[0] != 'n' && sql[0] != '\\' {
		return nil, nil, sql
	}
	token := MySQLNullToken{}
	if sql[:2] == "\\N" {
		token.value = "\\N"
		sql = sql[2:]
		return &token, nil, sql
	}
	keyword, err, sql := NewMySQLKeywordTokenByExpectKeywords(sql, []string{"NULL"})
	if err != nil {
		return nil, err, sql
	}
	if keyword != nil {
		token.value = keyword.Value()
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLNullToken) Value() string {
	return t.value
}

func (t *MySQLNullToken) Type() string {
	return "MySQLNullToken"
}

type MySQLNumericToken struct {
	value string
}

func NewMySQLNumericToken(sql string) (MySQLToken, error, string) {
	if sql[0] != '+' && sql[0] != '-' && sql[0] != '0' &&
		sql[0] != '1' && sql[0] != '2' && sql[0] != '3' &&
		sql[0] != '4' && sql[0] != '5' && sql[0] != '6' &&
		sql[0] != '7' && sql[0] != '8' && sql[0] != '9' &&
		sql[0] != '.' {
		return nil, nil, sql
	}
	numericRegex, err := regexp.Compile("(?i)^[+-]?(\\d+(\\.\\d*)?|\\.\\d+)(E[+-]?\\d+)?")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLNumericToken{}
	numericTest := numericRegex.MatchString(sql)
	if numericTest {
		token.value = numericRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLNumericToken) Value() string {
	return t.value
}

func (t *MySQLNumericToken) Type() string {
	return "MySQLNumericToken"
}

type MySQLOperatorToken struct {
	value string
}

var (
	operators = []string{
		"&&", "&", "||", "|", "~",
		"<<", "<=>", ">>", "<=", ">=",
		"<>", ">", "<", "!=", "!",
		"+", "-", "*", "/", "^",
		"%", "=", ":=", "(", ")",
		".",
	}
)

func NewMySQLOperatorToken(sql string) (MySQLToken, error, string) {
	if sql[0] != '&' && sql[0] != '|' && sql[0] != '~' &&
		sql[0] != '<' && sql[0] != '>' && sql[0] != '!' &&
		sql[0] != '+' && sql[0] != '-' && sql[0] != '*' &&
		sql[0] != '/' && sql[0] != '^' && sql[0] != '%' &&
		sql[0] != '=' && sql[0] != ':' && sql[0] != '(' &&
		sql[0] != ')' && sql[0] != '.' {
		return nil, nil, sql
	}
	token := MySQLOperatorToken{}
	for _, operator := range operators {
		if len(sql) >= len(operator) && sql[:len(operator)] == operator {
			token.value = operator
			sql = sql[len(token.value):]
			return &token, nil, sql
		}
	}
	return nil, nil, sql
}

func (t *MySQLOperatorToken) Value() string {
	return t.value
}

func (t *MySQLOperatorToken) Type() string {
	return "MySQLOperatorToken"
}

type MySQLQuotedIdentifierToken struct {
	value string
}

func NewMySQLQuotedIdentifierToken(sql string) (MySQLToken, error, string) {
	if sql[0] != '`' {
		return nil, nil, sql
	}
	regex, err := regexp.Compile("^`(``|[\u0001-\u005f\u0061-\uffff])+`")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLQuotedIdentifierToken{}
	test := regex.MatchString(sql)
	if test {
		token.value = regex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLQuotedIdentifierToken) Value() string {
	return t.value
}

func (t *MySQLQuotedIdentifierToken) Type() string {
	return "MySQLQuotedIdentifierToken"
}

type MySQLSpaceToken struct {
	value string
}

func NewMySQLSpaceToken(sql string) (MySQLToken, error, string) {
	if sql[0] != ' ' && sql[0] != '\t' && sql[0] != '\n' && sql[0] != '\r' {
		return nil, nil, sql
	}
	regex, err := regexp.Compile("^\\s+")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLSpaceToken{}
	test := regex.MatchString(sql)
	if test {
		token.value = regex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLSpaceToken) Value() string {
	return t.value
}

func (t *MySQLSpaceToken) Type() string {
	return "MySQLSpaceToken"
}

type MySQLStringToken struct {
	value string
}

func NewMySQLStringToken(sql string) (MySQLToken, error, string) {
	if sql[0] != 'N' && sql[0] != '"' && sql[0] != '\'' {
		return nil, nil, sql
	}
	singleQuotesRegex, err := regexp.Compile("(?i)^N?(''|'.*?[^\\\\]')")
	if err != nil {
		return nil, err, sql
	}
	doubleQuotesRegex, err := regexp.Compile("(?is)^N?(\"\"|\".*?[^\\\\]\")")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLStringToken{}
	singleTest := singleQuotesRegex.MatchString(sql)
	doubleTest := doubleQuotesRegex.MatchString(sql)
	if singleTest {
		r := singleQuotesRegex.FindStringSubmatch(sql)
		token.value = r[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if doubleTest {
		token.value = doubleQuotesRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLStringToken) Value() string {
	return t.value
}

func (t *MySQLStringToken) Type() string {
	return "MySQLStringToken"
}

type MySQLUnquotedIdentifierToken struct {
	value string
}

func NewMySQLUnquotedIdentifierToken(sql string) (MySQLToken, error, string) {
	regex, err := regexp.Compile("^\\b[0-9a-zA-Z$_\u0080-\uffff]+\\b")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLUnquotedIdentifierToken{}
	test := regex.MatchString(sql)
	if test {
		token.value = regex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLUnquotedIdentifierToken) Value() string {
	return t.value
}

func (t *MySQLUnquotedIdentifierToken) Type() string {
	return "MySQLUnquotedIdentifierToken"
}

type MySQLVariableToken struct {
	value string
}

func NewMySQLVariableToken(sql string) (MySQLToken, error, string) {
	if sql[0] != '@' {
		return nil, nil, sql
	}
	singleQuotesRegex, err := regexp.Compile("(?i)^@'(''|\\\\|\\'|[^'])*'")
	if err != nil {
		return nil, err, sql
	}
	doubleQuotesRegex, err := regexp.Compile("(?is)^@(\"\"|\".*?[^\\\\]\")")
	if err != nil {
		return nil, err, sql
	}
	quotesRegex, err := regexp.Compile("^@`(``|[\u0001-\u005f\u0061-\uffff])+`")
	if err != nil {
		return nil, err, sql
	}
	unquotesRegex, err := regexp.Compile("(?i)^@[0-9a-z_.$]+")
	if err != nil {
		return nil, err, sql
	}
	systemRegex, err := regexp.Compile("^@@(global\\.|session\\.)?[a-zA-Z-_]+")
	if err != nil {
		return nil, err, sql
	}

	token := MySQLVariableToken{}
	singleTest := singleQuotesRegex.MatchString(sql)
	doubleTest := doubleQuotesRegex.MatchString(sql)
	quotesTest := quotesRegex.MatchString(sql)
	unquotesTest := unquotesRegex.MatchString(sql)
	systemTest := systemRegex.MatchString(sql)
	if singleTest {
		token.value = singleQuotesRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if doubleTest {
		token.value = doubleQuotesRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if quotesTest {
		token.value = quotesRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if unquotesTest {
		token.value = unquotesRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	} else if systemTest {
		token.value = systemRegex.FindStringSubmatch(sql)[0]
		sql = sql[len(token.value):]
		return &token, nil, sql
	}
	return nil, nil, sql
}

func (t *MySQLVariableToken) Value() string {
	return t.value
}

func (t *MySQLVariableToken) Type() string {
	return "MySQLVariableToken"
}
