//
package fundmgr

import (
	"strconv"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/coupons"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/handler"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/match"
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
				setCache := atomic.SwapInt64(&match.SpeedSetCache, 0)
				getCache := atomic.SwapInt64(&match.SpeedGetCache, 0)
				missCache := atomic.SwapInt64(&handler.MissCache, 0)
				setRecordDB := atomic.SwapInt64(&match.SpeedSetDBRecord, 0)
				setStatusDB := atomic.SwapInt64(&match.SpeedSetDBStatus, 0)
				getRecordDB := atomic.SwapInt64(&match.SpeedGetDBRecord, 0)
				getStatusDB := atomic.SwapInt64(&match.SpeedGetDBStatus, 0)
				rankRecoverCnt := atomic.SwapInt32(&match.SpeedRecoverMatchRankRecord, 0)
				rankRecoverTime := atomic.SwapInt32(&match.SpeedRecoverMatchRankTime, 0)
				getMatchRankCache := atomic.SwapInt32(&match.GetMatchRankCache, 0)
				setMatchRankCache := atomic.SwapInt32(&match.SetMatchRankCache, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "set_match_cache", Value: strconv.Itoa(int(setCache))},
					logger.KeyValuePair{Key: "get_match_cache", Value: strconv.Itoa(int(getCache))},
					logger.KeyValuePair{Key: "miss_match_cache", Value: strconv.Itoa(int(missCache))},
				)
				logger.Stat(
					logger.KeyValuePair{Key: "set_db_match_record", Value: strconv.Itoa(int(setRecordDB))},
					logger.KeyValuePair{Key: "set_db_match_status", Value: strconv.Itoa(int(setStatusDB))},
					logger.KeyValuePair{Key: "get_db_match_record", Value: strconv.Itoa(int(getRecordDB))},
					logger.KeyValuePair{Key: "get_db_match_status", Value: strconv.Itoa(int(getStatusDB))},
				)
				logger.Stat(
					logger.KeyValuePair{Key: "get_rank_cache", Value: strconv.Itoa(int(getMatchRankCache))},
					logger.KeyValuePair{Key: "set_rank_cache", Value: strconv.Itoa(int(setMatchRankCache))},
					logger.KeyValuePair{Key: "rank_recover_time", Value: strconv.Itoa(int(rankRecoverTime))},
					logger.KeyValuePair{Key: "rank_recover_count", Value: strconv.Itoa(int(rankRecoverCnt))},
				)
				setLen, err := coupons.QueueLength(coupons.FUNDMGR_COUPONS_QUEUE)
				if err != nil {
					logger.Error("get queue: %v len error", coupons.FUNDMGR_COUPONS_QUEUE)
				}
				speedAcquireCoupons := atomic.SwapInt64(&coupons.SpeedAcquireCoupons, 0)
				AcquireCouponsSuccess := atomic.SwapInt64(&coupons.AcquireCouponsSuccess, 0)
				AcquireCouponsFailed := atomic.SwapInt64(&coupons.AcquireCouponsFailed, 0)
				QueryCouponsSuccess := atomic.SwapInt64(&coupons.QueryCouponsSuccess, 0)
				QueryCouponsFailed := atomic.SwapInt64(&coupons.QueryCouponsFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "coupons_queue_len", Value: strconv.Itoa(int(setLen))},
					logger.KeyValuePair{Key: "enqueue_acquire_coupons", Value: strconv.Itoa(int(speedAcquireCoupons))},
					logger.KeyValuePair{Key: "acquire_coupons_success", Value: strconv.Itoa(int(AcquireCouponsSuccess))},
					logger.KeyValuePair{Key: "acquire_coupons_failed", Value: strconv.Itoa(int(AcquireCouponsFailed))},
					logger.KeyValuePair{Key: "query_coupons_success", Value: strconv.Itoa(int(QueryCouponsSuccess))},
					logger.KeyValuePair{Key: "query_coupons_failed", Value: strconv.Itoa(int(QueryCouponsFailed))},
				)

				// rpc ops
				getMatchEventSuccess := atomic.SwapInt64(&handler.GetMatchEventSuccess, 0)
				getMatchEventFailed := atomic.SwapInt64(&handler.GetMatchEventFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetMatchEventSuccess", Value: strconv.Itoa(int(getMatchEventSuccess))},
					logger.KeyValuePair{Key: "GetMatchEventFailed", Value: strconv.Itoa(int(getMatchEventFailed))},
				)

				createMatchEventSuccess := atomic.SwapInt64(&handler.CreateMatchEventSuccess, 0)
				createMatchEventFailed := atomic.SwapInt64(&handler.CreateMatchEventFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "CreateMatchEventSuccess", Value: strconv.Itoa(int(createMatchEventSuccess))},
					logger.KeyValuePair{Key: "CreateMatchEventFailed", Value: strconv.Itoa(int(createMatchEventFailed))},
				)

				getUserMatchDonateSuccess := atomic.SwapInt64(&handler.GetUserMatchDonateSuccess, 0)
				getUserMatchDonateFailed := atomic.SwapInt64(&handler.GetUserMatchDonateFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetUserMatchDonateSuccess", Value: strconv.Itoa(int(getUserMatchDonateSuccess))},
					logger.KeyValuePair{Key: "GetUserMatchDonateFailed", Value: strconv.Itoa(int(getUserMatchDonateFailed))},
				)

				getUserWeekMatchRecordSuccess := atomic.SwapInt64(&handler.GetUserWeekMatchRecordSuccess, 0)
				getUserWeekMatchRecordFailed := atomic.SwapInt64(&handler.GetUserWeekMatchRecordFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetUserWeekMatchRecordSuccess", Value: strconv.Itoa(int(getUserWeekMatchRecordSuccess))},
					logger.KeyValuePair{Key: "GetUserWeekMatchRecordFailed", Value: strconv.Itoa(int(getUserWeekMatchRecordFailed))},
				)

				getUserMatchRecordByOffsetSuccess := atomic.SwapInt64(&handler.GetUserMatchRecordByOffsetSuccess, 0)
				getUserMatchRecordByOffsetFailed := atomic.SwapInt64(&handler.GetUserMatchRecordByOffsetFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetUserMatchRecordByOffsetSuccess", Value: strconv.Itoa(int(getUserMatchRecordByOffsetSuccess))},
					logger.KeyValuePair{Key: "GetUserMatchRecordByOffsetFailed", Value: strconv.Itoa(int(getUserMatchRecordByOffsetFailed))},
				)

				getUserTodayMatchSuccess := atomic.SwapInt64(&handler.GetUserTodayMatchSuccess, 0)
				getUserTodayMatchFailed := atomic.SwapInt64(&handler.GetUserTodayMatchFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetUserTodayMatchSuccess", Value: strconv.Itoa(int(getUserTodayMatchSuccess))},
					logger.KeyValuePair{Key: "GetUserTodayMatchFailed", Value: strconv.Itoa(int(getUserTodayMatchFailed))},
				)

				userMatchSuccess := atomic.SwapInt64(&handler.UserMatchSuccess, 0)
				userMatchFailed := atomic.SwapInt64(&handler.UserMatchFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "UserMatchSuccess", Value: strconv.Itoa(int(userMatchSuccess))},
					logger.KeyValuePair{Key: "UserMatchFailed", Value: strconv.Itoa(int(userMatchFailed))},
				)

				updateMatchSuccess := atomic.SwapInt64(&handler.UpdateMatchSuccess, 0)
				updateMatchFailed := atomic.SwapInt64(&handler.UpdateMatchFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "UpdateMatchSuccess", Value: strconv.Itoa(int(updateMatchSuccess))},
					logger.KeyValuePair{Key: "UpdateMatchFailed", Value: strconv.Itoa(int(updateMatchFailed))},
				)

				getActivityMatchInfoSuccess := atomic.SwapInt64(&handler.GetActivityMatchInfoSuccess, 0)
				getActivityMatchInfoFailed := atomic.SwapInt64(&handler.GetActivityMatchInfoFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetActivityMatchInfoSuccess", Value: strconv.Itoa(int(getActivityMatchInfoSuccess))},
					logger.KeyValuePair{Key: "GetActivityMatchInfoFailed", Value: strconv.Itoa(int(getActivityMatchInfoFailed))},
				)

				getActivityMatchRankSuccess := atomic.SwapInt64(&handler.GetActivityMatchRankSuccess, 0)
				getActivityMatchRankFailed := atomic.SwapInt64(&handler.GetActivityMatchRankFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetActivityMatchRankSuccess", Value: strconv.Itoa(int(getActivityMatchRankSuccess))},
					logger.KeyValuePair{Key: "GetActivityMatchRankFailed", Value: strconv.Itoa(int(getActivityMatchRankFailed))},
				)

				removeUserActivityMatchInfoSuccess := atomic.SwapInt64(&handler.RemoveUserActivityMatchInfoSuccess, 0)
				removeUserActivityMatchInfoFailed := atomic.SwapInt64(&handler.RemoveUserActivityMatchInfoFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "RemoveUserActivityMatchInfoSuccess", Value: strconv.Itoa(int(removeUserActivityMatchInfoSuccess))},
					logger.KeyValuePair{Key: "RemoveUserActivityMatchInfoFailed", Value: strconv.Itoa(int(removeUserActivityMatchInfoFailed))},
				)

				recoverUserActivityMatchRankSuccess := atomic.SwapInt64(&handler.RecoverUserActivityMatchRankSuccess, 0)
				recoverUserActivityMatchRankFailed := atomic.SwapInt64(&handler.RecoverUserActivityMatchRankFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "RecoverUserActivityMatchRankSuccess", Value: strconv.Itoa(int(recoverUserActivityMatchRankSuccess))},
					logger.KeyValuePair{Key: "RecoverUserActivityMatchRankFailed", Value: strconv.Itoa(int(recoverUserActivityMatchRankFailed))},
				)

				getCompanyMatchRankSuccess := atomic.SwapInt64(&handler.GetCompanyMatchRankSuccess, 0)
				getCompanyMatchRankFailed := atomic.SwapInt64(&handler.GetCompanyMatchRankFailed, 0)
				logger.Stat(
					logger.KeyValuePair{Key: "GetCompanyMatchRankSuccess", Value: strconv.Itoa(int(getCompanyMatchRankSuccess))},
					logger.KeyValuePair{Key: "GetCompanyMatchRankFailed", Value: strconv.Itoa(int(getCompanyMatchRankFailed))},
				)

				// reset loop
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
