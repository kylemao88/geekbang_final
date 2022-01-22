//
package stepmgr

import (
	"strconv"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/auto"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/handler"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/pk"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/steps"
)

const SLEEP_INTVAL = 60

var stopChan chan struct{}

func InitMonitorComponent() error {
	logger.Sys("Try InitMonitorComponent...")
	stopChan = make(chan struct{})
	// run step monitor
	go func() {
		logger.Sys("Monitor worker is start")
		t := time.NewTimer(time.Duration(SLEEP_INTVAL) * time.Second)
		for {
			select {
			case <-stopChan:
				logger.Sys("Monitor worker is stop")
				t.Stop()
				return
			case <-t.C:
				logger.Stat(logger.KeyValuePair{Key: "######################################", Value: ""})
				// data ops
				len, err := steps.QueueLength(steps.STEP_RECOVER_QUEUE)
				if err != nil {
					logger.Error("get queue: %v len error", steps.STEP_RECOVER_QUEUE)
				}
				setLen, err := steps.QueueLength(steps.STEP_SET_QUEUE)
				if err != nil {
					logger.Error("get queue: %v len error", steps.STEP_SET_QUEUE)
				}
				cntRecover := atomic.SwapInt64(&steps.SpeedRecover, 0)
				cntGetCache := atomic.SwapInt64(&steps.SpeedGetCache, 0)
				cntGetDB := atomic.SwapInt64(&steps.SpeedGetDB, 0)
				cntMiss := atomic.SwapInt64(&steps.MissCache, 0)
				cntPushUpdate := atomic.SwapInt64(&auto.SpeedPushUpdate, 0)
				cntUpdate := atomic.SwapInt64(&auto.SpeedUpdate, 0)
				cntSetDB := atomic.SwapInt64(&steps.SpeedSetDB, 0)
				getPkProfileCacheCnt := atomic.SwapInt64(&pk.GetPkProfileCacheCnt, 0)
				setPkProfileCacheCnt := atomic.SwapInt64(&pk.SetPkProfileCacheCnt, 0)
				getPkProfileDBCnt := atomic.SwapInt64(&pk.GetPkProfileDBCnt, 0)
				setPkProfileDBCnt := atomic.SwapInt64(&pk.SetPkProfileDBCnt, 0)
				getPkInteractCacheCnt := atomic.SwapInt64(&pk.GetPkInteractCacheCnt, 0)
				setPkInteractCacheCnt := atomic.SwapInt64(&pk.SetPkInteractCacheCnt, 0)
				getPkInteractDBCnt := atomic.SwapInt64(&pk.GetPkInteractDBCnt, 0)
				setPkInteractDBCnt := atomic.SwapInt64(&pk.SetPkInteractDBCnt, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "recover_queue_len", Value: strconv.Itoa(int(len))},
					logger.KeyValuePair{Key: "recover_cnt", Value: strconv.Itoa(int(cntRecover))},
				)
				logger.Stat(
					logger.KeyValuePair{Key: "set_queue_len", Value: strconv.Itoa(int(setLen))},
					logger.KeyValuePair{Key: "set_db_cnt", Value: strconv.Itoa(int(cntSetDB))},
				)
				logger.Stat(
					logger.KeyValuePair{Key: "auto_push_cnt", Value: strconv.Itoa(int(cntPushUpdate))},
					logger.KeyValuePair{Key: "auto_handle_cnt", Value: strconv.Itoa(int(cntUpdate))},
				)
				logger.Stat(
					logger.KeyValuePair{Key: "get_cache_cnt", Value: strconv.Itoa(int(cntGetCache))},
					logger.KeyValuePair{Key: "get_db_cnt", Value: strconv.Itoa(int(cntGetDB))},
					logger.KeyValuePair{Key: "cache_miss_cnt", Value: strconv.Itoa(int(cntMiss))},
				)
				logger.Stat(
					logger.KeyValuePair{Key: "get_pk_prof_cache", Value: strconv.Itoa(int(getPkProfileCacheCnt))},
					logger.KeyValuePair{Key: "set_pk_prof_cache", Value: strconv.Itoa(int(setPkProfileCacheCnt))},
					logger.KeyValuePair{Key: "get_pk_prof_db", Value: strconv.Itoa(int(getPkProfileDBCnt))},
					logger.KeyValuePair{Key: "set_pk_prof_db", Value: strconv.Itoa(int(setPkProfileDBCnt))},
				)
				logger.Stat(
					logger.KeyValuePair{Key: "get_pk_interact_cache", Value: strconv.Itoa(int(getPkInteractCacheCnt))},
					logger.KeyValuePair{Key: "set_pk_interact_cache", Value: strconv.Itoa(int(setPkInteractCacheCnt))},
					logger.KeyValuePair{Key: "get_pk_interact_db", Value: strconv.Itoa(int(getPkInteractDBCnt))},
					logger.KeyValuePair{Key: "set_pk_interact_db", Value: strconv.Itoa(int(setPkInteractDBCnt))},
				)

				// step rpc
				getUsersStepsSuccess := atomic.SwapInt64(&handler.GetUsersStepsSuccess, 0)
				getUsersStepsFailed := atomic.SwapInt64(&handler.GetUsersStepsFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetUsersStepsSuccess", Value: strconv.Itoa(int(getUsersStepsSuccess))},
					logger.KeyValuePair{Key: "GetUsersStepsFailed", Value: strconv.Itoa(int(getUsersStepsFailed))},
				)
				setUsersStepsSuccess := atomic.SwapInt64(&handler.SetUsersStepsSuccess, 0)
				setUsersStepsFailed := atomic.SwapInt64(&handler.SetUsersStepsFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "SetUsersStepsSuccess", Value: strconv.Itoa(int(setUsersStepsSuccess))},
					logger.KeyValuePair{Key: "SetUsersStepsFailed", Value: strconv.Itoa(int(setUsersStepsFailed))},
				)
				// pk rpc
				getUsersPkProfileSuccess := atomic.SwapInt64(&handler.GetUsersPkProfileSuccess, 0)
				getUsersPkProfileFailed := atomic.SwapInt64(&handler.GetUsersPkProfileFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetUsersPkProfileSuccess", Value: strconv.Itoa(int(getUsersPkProfileSuccess))},
					logger.KeyValuePair{Key: "GetUsersPkProfileFailed", Value: strconv.Itoa(int(getUsersPkProfileFailed))},
				)
				setUsersPkProfileSuccess := atomic.SwapInt64(&handler.SetUsersPkProfileSuccess, 0)
				setUsersPkProfileFailed := atomic.SwapInt64(&handler.SetUsersPkProfileFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "SetUsersPkProfileSuccess", Value: strconv.Itoa(int(setUsersPkProfileSuccess))},
					logger.KeyValuePair{Key: "SetUsersPkProfileFailed", Value: strconv.Itoa(int(setUsersPkProfileFailed))},
				)
				getPkInteractSuccess := atomic.SwapInt64(&handler.GetPkInteractSuccess, 0)
				getPkInteractFailed := atomic.SwapInt64(&handler.GetPkInteractFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetPkInteractSuccess", Value: strconv.Itoa(int(getPkInteractSuccess))},
					logger.KeyValuePair{Key: "GetPkInteractFailed", Value: strconv.Itoa(int(getPkInteractFailed))},
				)
				setPkInteractSuccess := atomic.SwapInt64(&handler.SetPkInteractSuccess, 0)
				setPkInteractFailed := atomic.SwapInt64(&handler.SetPkInteractFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "SetPkInteractSuccess", Value: strconv.Itoa(int(setPkInteractSuccess))},
					logger.KeyValuePair{Key: "SetPkInteractFailed", Value: strconv.Itoa(int(setPkInteractFailed))},
				)
				setActivePkSuccess := atomic.SwapInt64(&handler.SetActivePkSuccess, 0)
				setActivePkFailed := atomic.SwapInt64(&handler.SetActivePkFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "SetActivePkSuccess", Value: strconv.Itoa(int(setActivePkSuccess))},
					logger.KeyValuePair{Key: "SetActivePkFailed", Value: strconv.Itoa(int(setActivePkFailed))},
				)
				setResponsePkSuccess := atomic.SwapInt64(&handler.SetResponsePkSuccess, 0)
				setResponsePkFailed := atomic.SwapInt64(&handler.SetResponsePkFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "SetResponsePkSuccess", Value: strconv.Itoa(int(setResponsePkSuccess))},
					logger.KeyValuePair{Key: "SetResponsePkFailed", Value: strconv.Itoa(int(setResponsePkFailed))},
				)
				getPassivePkSuccess := atomic.SwapInt64(&handler.GetPassivePkSuccess, 0)
				getPassivePkFailed := atomic.SwapInt64(&handler.GetPassivePkFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetPassivePkSuccess", Value: strconv.Itoa(int(getPassivePkSuccess))},
					logger.KeyValuePair{Key: "GetPassivePkFailed", Value: strconv.Itoa(int(getPassivePkFailed))},
				)
				getPkNotificationSuccess := atomic.SwapInt64(&handler.GetPkNotificationSuccess, 0)
				getPkNotificationFailed := atomic.SwapInt64(&handler.GetPkNotificationFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetPkNotificationSuccess", Value: strconv.Itoa(int(getPkNotificationSuccess))},
					logger.KeyValuePair{Key: "GetPkNotificationFailed", Value: strconv.Itoa(int(getPkNotificationFailed))},
				)

				t.Reset(time.Duration(SLEEP_INTVAL) * time.Second)
			}
		}
	}()
	return nil
}

func CloseMonitorComponent() {
	logger.Sys("Try CloseMonitorComponent...")
	close(stopChan)
	time.Sleep(1 * time.Second)
}
