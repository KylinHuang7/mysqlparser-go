package mysqlparser_go

import (
	"fmt"
	"strings"
)

type LogLevel int32

const (
	LogLevelError  LogLevel = 1
	LogLevelNotice LogLevel = 2
	LogLevelInfo   LogLevel = 3
	LogLevelDebug  LogLevel = 4
)

var CurrentLogLevel = LogLevelNotice
var LogFunc = fmt.Println

func verbose(content string, level LogLevel) {
	if LogFunc != nil {
		if level <= CurrentLogLevel {
			LogFunc(content)
		}
	}
}

type ObjectType int

const (
	UNKNOWN   ObjectType = 0
	STATEMENT ObjectType = 1
	COMPONENT ObjectType = 2
	TOKEN     ObjectType = 3
)

func GetObjectType(t string) ObjectType {
	if strings.HasSuffix(t, "Statement") {
		return STATEMENT
	} else if strings.HasSuffix(t, "Component") {
		return COMPONENT
	} else if strings.HasSuffix(t, "Token") {
		return TOKEN
	}
	return UNKNOWN
}

type MySQLObject interface {
	Type() string
	Value() string
}

const FinalStatus = 999

type FsmMap struct {
	StartStatus  []int
	AcceptObject string
	AcceptValue  string
	EndStatus    int
}

func (m *FsmMap) ToString() string {
	return fmt.Sprintf("%+v --%s(%s)-> [%d]", m.StartStatus, m.AcceptObject, m.AcceptValue, m.EndStatus)
}
