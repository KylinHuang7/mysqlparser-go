package mysqlparser_go

import (
	"fmt"
	"testing"
)

func Test_Parser(t *testing.T) {
	sqlmap := map[string]bool{
		"UPDATE DB_Ad_43.Tbl_AdGroup_1 SET `FUserStatus`=10,`FLastModTime`=1661181610,`FColdStartAudienceIdSet`='',`FExpandTargetingRule`='',`FLastModByDeveloperAppId`=1110260112,`FLastModByUserId`=25699284 WHERE FAId=6356055783;INSERT INTO DB_Ad_43.Tbl_DiTraceLog_1 (`FSeqID`,`FUId`,`FDbName`,`FTableName`,`FOperationType`,`FOperation`,`FLogStatus`,`FCreatedTime`,`FLastModTime`,`FDiReturnCode`,`FDiEventNo`,`FDiErrInfo`) VALUES (13290762304,24453143,'DB_Ad_43','DB_Ad_43.Tbl_AdGroup_1',1,'{\\\\\\\"operation_type\\\\\\\":1,\\\\\\\"table\\\\\\\":\\\\\\\"DB_Ad_.Tbl_AdGroup_\\\\\\\",\\\\\\\"row\\\\\\\":{\\\\\\\"names\\\\\\\":[\\\\\\\"FUserStatus\\\\\\\",\\\\\\\"FLastModTime\\\\\\\",\\\\\\\"FColdStartAudienceIdSet\\\\\\\",\\\\\\\"FExpandTargetingRule\\\\\\\",\\\\\\\"FLastModByDeveloperAppId\\\\\\\",\\\\\\\"FLastModByUserId\\\\\\\"],\\\\\\\"values\\\\\\\":[{\\\\\\\"uint_value\\\\\\\":10,\\\\\\\"type\\\\\\\":2},{\\\\\\\"uint_value\\\\\\\":1661181610,\\\\\\\"type\\\\\\\":2},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"uint_value\\\\\\\":1110260112,\\\\\\\"type\\\\\\\":2},{\\\\\\\"int_value\\\\\\\":25699284,\\\\\\\"type\\\\\\\":1}],\\\\\\\"old_values\\\\\\\":[{\\\\\\\"uint_value\\\\\\\":1,\\\\\\\"type\\\\\\\":2},{\\\\\\\"uint_value\\\\\\\":1661177504,\\\\\\\"type\\\\\\\":2},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"uint_value\\\\\\\":1110741802,\\\\\\\"type\\\\\\\":2},{\\\\\\\"type\\\\\\\":1}]},\\\\\\\"where_args\\\\\\\":{\\\\\\\"condition\\\\\\\":\\\\\\\"FAId=?\\\\\\\",\\\\\\\"condition_args\\\\\\\":[{\\\\\\\"int_value\\\\\\\":6356055783,\\\\\\\"type\\\\\\\":1}]},\\\\\\\"divide_key\\\\\\\":24453143,\\\\\\\"primary_keys\\\\\\\":[\\\\\\\"FAId\\\\\\\"],\\\\\\\"context\\\\\\\":{\\\\\\\"protocol_type\\\\\\\":3,\\\\\\\"user_command\\\\\\\":3,\\\\\\\"operation_client\\\\\\\":1,\\\\\\\"operator_role\\\\\\\":1,\\\\\\\"operation_action\\\\\\\":2,\\\\\\\"frontend_operator\\\\\\\":\\\\\\\"1704907017\\\\\\\",\\\\\\\"frontend_operator_type\\\\\\\":1,\\\\\\\"frontend_operation_object\\\\\\\":3,\\\\\\\"trace_id\\\\\\\":\\\\\\\"b8719df6-8bb8-4999-e863-a7b19ed0462e\\\\\\\",\\\\\\\"operator_name\\\\\\\":\\\\\\\"1704907017\\\\\\\",\\\\\\\"operator_type\\\\\\\":\\\\\\\"qq\\\\\\\",\\\\\\\"operator_platform\\\\\\\":\\\\\\\"1002\\\\\\\"},\\\\\\\"route_key\\\\\\\":\\\\\\\"FUId\\\\\\\",\\\\\\\"route_key_values\\\\\\\":[{\\\\\\\"uint_value\\\\\\\":24453143,\\\\\\\"type\\\\\\\":2}]}',255,1661181610,1661181610,1,0,'')": true,
		"UPDATE DB_Ad_43.Tbl_AdGroup_1 SET `FUserStatus`=10,`FLastModTime`=1661181610,`FColdStartAudienceIdSet`='',`FExpandTargetingRule`='',`FLastModByDeveloperAppId`=1110260112,`FLastModByUserId`=25699284 WHERE FAId=6356055783;INSERT INTO DB_Ad_43.Tbl_DiTraceLog_1 (`FSeqID`,`FUId`,`FDbName`,`FTableName`,`FOperationType`,`FOperation`,`FLogStatus`,`FCreatedTime`,`FLastModTime`,`FDiReturnCode`,`FDiEventNo`,`FDiErrInfo`) VALUES (13290762304,24453143,'DB_Ad_43','DB_Ad_43.Tbl_AdGroup_1',1,'{\\\\\\\"operation_type\\\\\\\":1,\\\\\\\"table\\\\\\\":\\\\\\\"DB_Ad_.Tbl_AdGroup_\\\\\\\",\\\\\\\"row\\\\\\\":{\\\\\\\"names\\\\\\\":[\\\\\\\"FUserStatus\\\\\\\",\\\\\\\"FLastModTime\\\\\\\",\\\\\\\"FColdStartAudienceIdSet\\\\\\\",\\\\\\\"FExpandTargetingRule\\\\\\\",\\\\\\\"FLastModByDeveloperAppId\\\\\\\",\\\\\\\"FLastModByUserId\\\\\\\"],\\\\\\\"values\\\\\\\":[{\\\\\\\"uint_value\\\\\\\":10,\\\\\\\"type\\\\\\\":2},{\\\\\\\"uint_value\\\\\\\":1661181610,\\\\\\\"type\\\\\\\":2},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"uint_value\\\\\\\":1110260112,\\\\\\\"type\\\\\\\":2},{\\\\\\\"int_value\\\\\\\":25699284,\\\\\\\"type\\\\\\\":1}],\\\\\\\"old_values\\\\\\\":[{\\\\\\\"uint_value\\\\\\\":1,\\\\\\\"type\\\\\\\":2},{\\\\\\\"uint_value\\\\\\\":1661177504,\\\\\\\"type\\\\\\\":2},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"type\\\\\\\":4},{\\\\\\\"uint_value\\\\\\\":1110741802,\\\\\\\"type\\\\\\\":2},{\\\\\\\"type\\\\\\\":1}]},\\\\\\\"where_args\\\\\\\":{\\\\\\\"condition\\\\\\\":\\\\\\\"FAId=?\\\\\\\",\\\\\\\"condition_args\\\\\\\":[{\\\\\\\"int_value\\\\\\\":6356055783,\\\\\\\"type\\\\\\\":1}]},\\\\\\\"divide_key\\\\\\\":24453143,\\\\\\\"primary_keys\\\\\\\":[\\\\\\\"FAId\\\\\\\"],\\\\\\\"context\\\\\\\":{\\\\\\\"protocol_type\\\\\\\":3,\\\\\\\"user_command\\\\\\\":3,\\\\\\\"operation_client\\\\\\\":1,\\\\\\\"operator_role\\\\\\\":1,\\\\\\\"operation_action\\\\\\\":2,\\\\\\\"frontend_operator\\\\\\\":\\\\\\\"1704907017\\\\\\\",\\\\\\\"frontend_operator_type\\\\\\\":1,\\\\\\\"frontend_operation_object\\\\\\\":3,\\\\\\\"trace_id\\\\\\\":\\\\\\\"b8719df6-8bb8-4999-e863-a7b19ed0462e\\\\\\\",\\\\\\\"operator_name\\\\\\\":\\\\\\\"1704907017\\\\\\\",\\\\\\\"operator_type\\\\\\\":\\\\\\\"qq\\\\\\\",\\\\\\\"operator_platform\\\\\\\":\\\\\\\"1002\\\\\\\"},\\\\\\\"route_key\\\\\\\":\\\\\\\"FUId\\\\\\\",\\\\\\\"route_key_values\\\\\\\":[{\\\\\\\"uint_value\\\\\\\":24453143,\\\\\\\"type\\\\\\\":2}]}',255,1661181610,1661181610,":        false,
	}
	for sql, result := range sqlmap {
		statementList, err := Parse(sql)
		if result {
			if err != nil {
				t.Errorf("Error: %+v", err)
			}
			if statementList != nil {
				fmt.Printf("%+v\n", statementList)
			}
		} else {
			if err == nil {
				t.Errorf("Error")
			}
		}
	}
}
