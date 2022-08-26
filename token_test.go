package mysqlparser_go

import (
	"fmt"
	"testing"
)

func tokenTestTemplate(t *testing.T, f func(sql string) (MySQLToken, error, string), sqlmap map[string]string) {
	for sql, result := range sqlmap {
		newToken, err, left := f(sql)
		if err != nil {
			t.Errorf("Error: %+v", err)
		}
		if newToken != nil {
			if result != newToken.Value() {
				t.Errorf("Respect: %+v, Got: %+v", result, newToken.Value())
			}
		} else {
			if result != "" {
				t.Errorf("Respect: %+v, Got empty", result)
			}
		}
		fmt.Printf("Left: %s\n", left)
	}
}

func Test_Bit(t *testing.T) {
	sqlmap := map[string]string{
		"0b101":   "0b101",
		"B'1011'": "B'1011'",
		"0b1012":  "0b101",
	}
	tokenTestTemplate(t, NewMySQLBitToken, sqlmap)
}

func Test_Comment(t *testing.T) {
	sqlmap := map[string]string{
		"#abcd":       "#abcd",
		"abcd#abcd":   "",
		"-- abcd":     "-- abcd",
		"/*abcd*/efg": "/*abcd*/",
	}
	tokenTestTemplate(t, NewMySQLCommentToken, sqlmap)
}

func Test_Delimiter(t *testing.T) {
	sqlmap := map[string]string{
		";abc": ";",
		",abc": ",",
	}
	tokenTestTemplate(t, NewMySQLDelimiterToken, sqlmap)
}

func Test_Hexadecimal(t *testing.T) {
	sqlmap := map[string]string{
		"0xa7cd":  "0xa7cd",
		"X'89a1'": "X'89a1'",
		"0xa7beq": "0xa7be",
	}
	tokenTestTemplate(t, NewMySQLHexadecimalToken, sqlmap)
}

func Test_Keyword(t *testing.T) {
	sqlmap := map[string]string{
		"SELECT":       "SELECT",
		"update sasa":  "UPDATE",
		"updateabc":    "",
		"test UPDATE ": "",
	}
	tokenTestTemplate(t, NewMySQLKeywordToken, sqlmap)
}

func Test_Null(t *testing.T) {
	sqlmap := map[string]string{
		"\\N":    "\\N",
		"null":   "NULL",
		"NULLIF": "",
	}
	tokenTestTemplate(t, NewMySQLNullToken, sqlmap)
}

func Test_Numeric(t *testing.T) {
	sqlmap := map[string]string{
		"1234":     "1234",
		"1234abc":  "1234",
		"-123.819": "-123.819",
	}
	tokenTestTemplate(t, NewMySQLNumericToken, sqlmap)
}

func Test_Operator(t *testing.T) {
	sqlmap := map[string]string{
		"+ 1":        "+",
		"(abcd + 1)": "(",
		"|| a":       "||",
		"|a":         "|",
	}
	tokenTestTemplate(t, NewMySQLOperatorToken, sqlmap)
}

func Test_Quoted_Identifier(t *testing.T) {
	sqlmap := map[string]string{
		"`abc`": "`abc`",
		"abc":   "",
		"`你好`啊": "`你好`",
	}
	tokenTestTemplate(t, NewMySQLQuotedIdentifierToken, sqlmap)
}

func Test_Space(t *testing.T) {
	sqlmap := map[string]string{
		" abcc":    " ",
		"\t\tabc":  "\t\t",
		"\n \tabc": "\n \t",
	}
	tokenTestTemplate(t, NewMySQLSpaceToken, sqlmap)
}

func Test_String(t *testing.T) {
	sqlmap := map[string]string{
		"\"\"":       "\"\"",
		"\"abc\"":    "\"abc\"",
		"\"abc\"def": "\"abc\"",
		"'abc'":      "'abc'",
		"'abc'd":     "'abc'",
		"'',''":      "''",
		"'\\'',''":   "'\\''",
	}
	tokenTestTemplate(t, NewMySQLStringToken, sqlmap)
}

func Test_Unquoted_Identifier(t *testing.T) {
	sqlmap := map[string]string{
		"abc":   "abc",
		"abc d": "abc",
	}
	tokenTestTemplate(t, NewMySQLUnquotedIdentifierToken, sqlmap)
}

func Test_Variable(t *testing.T) {
	sqlmap := map[string]string{
		"@abc":         "@abc",
		"@@global.abc": "@@global.abc",
		"@@abcd.xyz":   "@@abcd",
	}
	tokenTestTemplate(t, NewMySQLVariableToken, sqlmap)
}
