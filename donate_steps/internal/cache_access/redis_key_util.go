//
//

package cache_access

import (
	"fmt"
)

const OidToUnidHashKey = "donate_steps_oid_unid_map"

// user friend key prefix
const (
	FRIEND_LIST_ZSET_KEY = "yqz:zset:friend_list"
)

func GetFriendListKey(oid string) string {
	return fmt.Sprintf("%s:%s", FRIEND_LIST_ZSET_KEY, oid)
}

