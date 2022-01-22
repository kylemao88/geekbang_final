//
//

package common

const (
	SUCCESS                     = 0
	SESSION_ERROR               = 1000
	PARAMS_ERROR                = 1001
	INNTER_ERROR                = 1002
	NO_PERMISSION               = 1003
	SESSION_CODE                = 1004
	DECODE_ERROR                = 1005
	EVENT_END                   = 1006
	RE_DONATION                 = 1007
	JOINED                      = 1008
	NOT_JOIN                    = 1009
	RETRY                       = 1010
	DB_ERROR                    = 2000
	NO_MONEY                    = 2001
	HAVE_DIRTY                  = 2002
	TOPLIMIT                    = 2003
	EVENT_END_NOT_WEEKLY        = 2004
	EVENT_START_NOT_WEEKLY      = 2005
	NO_BIND                     = 2006
	JOINMAX                     = 2007
	ORDERFINISH                 = 2008
	EMPTY_DATA                  = 2009
	NOT_ACTIVE                  = 2010
	ONE_DAY_STEP_OVER_LIMIT     = 2011
	HAVE_ENVELOPE               = 2012
	TRY_AGAIN                   = 2013
	NOT_RECORD                  = 2014
	USER_MAX_RED                = 2015
	NOT_EXIST_PK                = 2016
	RED_PACKET_EXPIRED          = 2017
	NOT_MSG_TEMPLE              = 2018
	TEAM_USER_LIMIT             = 2019
	MAX_LIMIT                   = 2020
	CREATE_ACTIVITY_LIMIT       = 2021
	OPS_FOBBID                  = 2022
	OPS_START_TIME_DIFF         = 2023
	OPS_OVER_TIME               = 2024
	OPS_CREATE_NOT_SELF         = 2025
	JOIN_ACTIVITY_LIMIT         = 2030
	JOIN_TEAM_MEMBER_LIMIT      = 2031
	DEL_ACTIVITY_CREATOR        = 2032
	DEL_ACTIVITY_USER_FORBIDDEN = 2033
	JOIN_ACTIVITY_NO_PERMISSION = 2034
	NO_MATCH_ITEM               = 2101
	INVALID_ORDER_NO            = 2102
	QRY_COUPONS_LIST_FAIL       = 2103
	WX_NO_PERM                  = 3000
)

var ErrCodeMsgMap = map[int]string{
	SUCCESS:                     "suucess",
	SESSION_ERROR:               "session error",
	PARAMS_ERROR:                "params error",
	INNTER_ERROR:                "internal error",
	NO_PERMISSION:               "No permission to modify",
	SESSION_CODE:                "session code error",
	DECODE_ERROR:                "decode wx error",
	EVENT_END:                   "event is end",
	RE_DONATION:                 "repeat donation",
	JOINED:                      "joined event",
	NOT_JOIN:                    "not join event",
	RETRY:                       "retry",
	DB_ERROR:                    "db error",
	NO_MONEY:                    "no money",
	HAVE_DIRTY:                  "dirty comment",
	TOPLIMIT:                    "top over limit",
	EVENT_END_NOT_WEEKLY:        "event end not weekly data",
	EVENT_START_NOT_WEEKLY:      "event start not weekly data",
	NO_BIND:                     "not bind, cant unbind",
	JOINMAX:                     "join event max",
	ORDERFINISH:                 "order finish",
	EMPTY_DATA:                  "empty data",
	NOT_ACTIVE:                  "event not active",
	ONE_DAY_STEP_OVER_LIMIT:     "one day step over limit",
	HAVE_ENVELOPE:               "red have envelope",
	TRY_AGAIN:                   "try again",
	NOT_RECORD:                  "not red record",
	USER_MAX_RED:                "user max red packet",
	NOT_EXIST_PK:                "not exist pk info",
	RED_PACKET_EXPIRED:          "red packet expired",
	TEAM_USER_LIMIT:             "team user over limit",
	MAX_LIMIT:                   "max member limit",
	CREATE_ACTIVITY_LIMIT:       "user create activity over limit",
	WX_NO_PERM:                  "need auth again",
	JOIN_ACTIVITY_LIMIT:         "join activity over limit",
	JOIN_TEAM_MEMBER_LIMIT:      "join team over team member limit",
	JOIN_ACTIVITY_NO_PERMISSION: "No permission to join activity",
	DEL_ACTIVITY_CREATOR:        "delete activity creator forbidden",
	DEL_ACTIVITY_USER_FORBIDDEN: "delete activity user forbidden as status not match",
	INVALID_ORDER_NO:            "invalid conpons order_no",
	QRY_COUPONS_LIST_FAIL:       "query coupons list failed",
}
