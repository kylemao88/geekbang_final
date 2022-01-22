//
package cache_access

import "fmt"

const (
	WX_STEP_APPID = "wxff244f6b82a094d2"

	// user profile cache, use hashmap
	YQZ_USER_PROFILE_KEY            = "YQZ_USER_PROFILE"
	YQZ_USER_PROFILE_YQZID          = "PROFILE_YQZID"
	YQZ_USER_PROFILE_SEND_LEAF      = "PROFILE_SEND_LEAF"
	YQZ_USER_PROFILE_DONATE_FLAG    = "PROFILE_DONATE_FLAG"    // 0 表示未捐过步，1表示捐过步
	YQZ_USER_PROFILE_TEAM_FLAG      = "PROFILE_TEAM_FLAG"      // 1 表示没有加入活动，2表示加入活动
	YQZ_USER_PROFILE_AUTO_STEP_FLAG = "PROFILE_AUTO_STEP_FLAG" // 1 表示不支持，2 表示支持
	YQZ_USER_PROFILE_AUTO_STEP_TIME = "PROFILE_AUTO_STEP_TIME"
	YQZ_USER_PROFILE_PK_OBJ         = "PROFILE_PK_OBJ"

	// user subscribe cache, use hashmap
	YQZ_USER_SUBSCRIBE_KEY = "YQZ_USER_SUBSCRIBE"

	// user team list, use zset
	YQZ_USER_TEAM_KEY = "YQZ_USER_TEAM_LIST"

	// team stat, use zset
	YQZ_TEAM_STEP_KEY = "YQZ_TEAM_USER_STAT"
	YQZ_TEAM_LEAF_KEY = "YQZ_TEAM_LEAF_STAT"
)

type UserProfile struct {
	Unin     string
	Leaf     int
	Donated  int
	Joined   int
	AutoFlag int
	AutoTime string
}

type TeamUserStat struct {
	Oid   string
	Score int
}

type TeamIDAndTime struct {
	TeamID string
	Time   int
}

func FormatUserProfileKey(oid string) string {
	return fmt.Sprintf("%v:%v", YQZ_USER_PROFILE_KEY, oid)
}

func FormatTeamStepStatKey(teamID string) string {
	return fmt.Sprintf("%v:%v", YQZ_TEAM_STEP_KEY, teamID)
}

func FormatTeamLeafStatKey(teamID string) string {
	return fmt.Sprintf("%v:%v", YQZ_TEAM_LEAF_KEY, teamID)
}

func FormatUserTeamKey(oid string) string {
	return fmt.Sprintf("%v:%v", YQZ_USER_TEAM_KEY, oid)
}

func FormatUserSubKey(oid string) string {
	return fmt.Sprintf("%v:%v", YQZ_USER_SUBSCRIBE_KEY, oid)
}
