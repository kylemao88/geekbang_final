//
package steps

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/config"
	"github.com/go-redis/redis"
)

const STEP_USER_KEY_PREFIX = "yqz:stepmgr:user"

var stopChan chan struct{}
var stopSetChan chan struct{}

// for monitor
var (
	SpeedGetCache int64 = 0
	SpeedGetDB    int64 = 0
	SpeedSetDB    int64 = 0
	MissCache     int64 = 0
)

type UserStepsRecord struct {
	FUserId       string `json:"f_user_id"`
	FStep         int64  `json:"f_step"`
	FUpdateMethod int    `json:"f_update_method"`
	FDate         string `json:"f_date"`
	FCreateTime   string `json:"f_create_time"`
	FModifyTime   string `json:"f_modify_time"`
}

var (
	cdbClient   dbclient.DBClient
	redisClient *gyredis.RedisClient
)

// InitStepComponent ...
func InitStepComponent(cdbCli dbclient.DBClient, redisCli *gyredis.RedisClient, stepConf config.StepConfig) error {
	logger.Sys("Try InitStepComponent...")
	stopChan = make(chan struct{})
	stopSetChan = make(chan struct{})
	cdbClient = cdbCli
	redisClient = redisCli
	err := runRecoverWorker(stepConf.RecoverCount, stepConf.RecoverIntervalMs)
	if err != nil {
		return err
	}
	err = runSetWorker(stepConf.RecoverCount, stepConf.RecoverIntervalMs)
	if err != nil {
		return err
	}
	return err
}

// CloseStepComponent ...
func CloseStepComponent() {
	logger.Sys("Try CloseStepComponent...")
	closeRecoverWorker()
	closeSetWorker()
	time.Sleep(1 * time.Second)
}

// GetUserSteps return most recent 30 days steps
func GetUserSteps(userID string) (*metadata.UserSteps, error) {
	result, err := GetUsersStepsFromCache([]string{userID})
	if err != nil {
		if errors.IsStepMgrIncompleteError(err) {
			return nil, errors.NewRedisNilError()
		}
		return nil, err
	}
	return result[userID], nil
}

// GetUsersSteps return most recent 30 days steps
func GetUsersSteps(userIDs []string) (map[string]*metadata.UserSteps, error) {
	result, err := GetUsersStepsFromCache(userIDs)
	if err != nil && !errors.IsStepMgrIncompleteError(err) {
		return nil, err
	}
	return result, err
}

// GetUsersRangeSteps return range days steps
func GetUsersRangeSteps(userIDs []string, start, end string) (map[string]*metadata.UserSteps, error) {
	defer util.GetUsedTime(fmt.Sprintf("%v - count: %v range:[%v-%v]", util.GetCallee(), len(userIDs), start, end))()
	// check if range is in 30 days
	startTime, err := time.ParseInLocation("2006-01-02", start, util.Loc)
	if err != nil {
		return nil, errors.NewStepMgrParamInvalid(errors.WithMsg("%v - start-end: [%v-%v]", util.GetCallee(), start, end))
	}
	endTime, err := time.ParseInLocation("2006-01-02", end, util.Loc)
	if err != nil {
		return nil, errors.NewStepMgrParamInvalid(errors.WithMsg("%v - start-end: [%v-%v]", util.GetCallee(), start, end))
	}
	currentTime := time.Now()
	todayTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, util.Loc).Unix()
	duration, _ := time.ParseDuration("-720h")
	delta := time.Now().Add(duration)
	deltaTime := time.Date(delta.Year(), delta.Month(), delta.Day(), 0, 0, 0, 0, util.Loc).Unix()
	// get value from cache or db
	var result = make(map[string]*metadata.UserSteps)
	var resultErr error
	if startTime.Unix() >= deltaTime && endTime.Unix() <= todayTime {
		// if in recently 30 days we just get from redis
		result, resultErr = GetUsersSteps(userIDs)
		if resultErr != nil && !errors.IsStepMgrIncompleteError(resultErr) {
			return nil, resultErr
		}
		for _, steps := range result {
			for date := range steps.Steps {
				dateTime, err := time.ParseInLocation("2006-01-02", date, util.Loc)
				if err != nil {
					return nil, errors.NewStepMgrInternalError(errors.WithMsg("%v - date: %v", util.GetCallee(), date))
				}
				in := inTimeSpan(startTime, endTime, dateTime)
				if !in {
					if dateTime != startTime && dateTime != endTime {
						delete(steps.Steps, date)
					}
				}
			}
		}
	} else {
		// if not in recently 30 days we just get from db, but here should not call frequently
		result, resultErr = GetUsersStepsByDateRangeFromDB(userIDs, start, end)
		if resultErr != nil && !errors.IsStepMgrIncompleteError(resultErr) {
			return nil, resultErr
		}
		logger.Info("%v - users: %v start-end: [%v-%v] out of range 30 days", util.GetCallee(), userIDs, start, end)
	}
	return result, resultErr
}

func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

// GetUsersStepsFromCache ...
func GetUsersStepsFromCache(userIDs []string) (map[string]*metadata.UserSteps, error) {
	defer util.GetUsedTime(fmt.Sprintf("%v - count: %v", util.GetCallee(), len(userIDs)))()
	atomic.AddInt64(&SpeedGetCache, int64(len(userIDs))) //加操作
	// use pipeline get all user's step from cache
	var incomplete = []string{}
	var result = make(map[string]*metadata.UserSteps)
	for _, user := range userIDs {
		userKey := fmt.Sprintf("%s:%s", STEP_USER_KEY_PREFIX, user)
		val, err := redisClient.HGetAll(userKey).Result()
		// unExpect error
		if err != nil && err != redis.Nil {
			return nil, errors.NewRedisClientError(errors.WithMsg("%v - redis Get error, error = %v, key = %s",
				util.GetCallee(), err, userKey))
		}
		if err == redis.Nil || len(val) == 0 {
			logger.Info("%v - redis Get key = %s not exist", util.GetCallee(), userKey)
			incomplete = append(incomplete, user)
			// here we need call other to recover redis
			ret, err := pushRecoverUser(user)
			if err != nil {
				logger.Error("PushRecoverUser %v err: %v", user, err)
			} else if ret != 1 {
				logger.Info("PushRecoverUser %v already exist ", user)
			} else {
				logger.Info("PushRecoverUser %v success", user)
			}
			atomic.AddInt64(&MissCache, 1) //加操作
			continue
		}
		for date, value := range val {
			step, err := strconv.Atoi(value)
			if err != nil {
				return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err))
			}
			if value, ok := result[user]; !ok {
				userStep := &metadata.UserSteps{
					Oid: user,
					Steps: map[string]int32{
						date: int32(step),
					},
				}
				result[user] = userStep
			} else {
				value.Steps[date] = int32(step)
			}
		}
	}
	var err error = nil
	if len(incomplete) != 0 {
		err = errors.NewStepMgrIncompleteError(errors.WithMsg("incomplete result: %v", incomplete))
	}
	logger.Debug("%v - %v user: %v, get %v result: %v, leak: %v", util.GetCallee(), len(userIDs), userIDs, len(result), result, incomplete)
	return result, err
}

// GetUsersStepsByDateRangeFromDB ...
func GetUsersStepsByDateRangeFromDB(userIDs []string, start string, end string) (map[string]*metadata.UserSteps, error) {
	defer util.GetUsedTime(fmt.Sprintf("%v - count: %v range: %v-%v", util.GetCallee(), len(userIDs), start, end))()
	atomic.AddInt64(&SpeedGetDB, int64(len(userIDs))) //加操作
	sql_str := "select f_user_id, f_step, f_update_method, f_date, f_create_time, f_modify_time from t_step where f_user_id in (?) and f_date between ? and ?"
	var usersStr string
	for _, user := range userIDs {
		usersStr += user + ","
	}
	usersStr = fmt.Sprintf("%v", strings.TrimSuffix(usersStr, ","))
	args := make([]interface{}, 0)
	args = append(args, usersStr, start, end)
	res, err := cdbClient.Query(sql_str, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - QueryDB error: %v,", util.GetCallee(), err))
	}
	// no match record just return empty record
	if len(res) == 0 {
		logger.Info("%v - users: %v, date from: %v - %v get no step record", util.GetCallee(), userIDs, start, end)
		return nil, errors.NewStepMgrIncompleteError(errors.WithMsg("leak result count: %v", len(userIDs)))
	}
	// format result
	var result = make(map[string]*metadata.UserSteps)
	for _, value := range res {
		step, err := strconv.Atoi(value["f_step"])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value["f_step"]))
		}
		user := value["f_user_id"]
		date := strings.TrimSuffix(value["f_date"], " 00:00:00")
		if value, ok := result[user]; !ok {
			userStep := &metadata.UserSteps{
				Oid: user,
				Steps: map[string]int32{
					date: int32(step),
				},
			}
			result[user] = userStep
		} else {
			value.Steps[date] = int32(step)
		}
	}
	err = nil
	if len(result) != len(userIDs) {
		err = errors.NewStepMgrIncompleteError(errors.WithMsg("leak result count: %v", len(userIDs)-len(result)))
	}
	logger.Debug("%v - %v user: %v, get %v result: %v", util.GetCallee(), len(userIDs), userIDs, len(result), result)
	return result, err
}

// SetUsersSteps ...
func SetUsersSteps(steps map[string]*metadata.UserSteps, background bool) error {
	// defer util.GetUsedTime(fmt.Sprintf("%v - count: %v back: %v", util.GetCallee(), len(steps), background))()
	// limit record count insert to db
	if len(steps) > config.GetConfig().StepConf.DBUpdateBatch {
		var batchSteps = make(map[string]*metadata.UserSteps)
		var count = 0
		var batchCount = 0
		for k, v := range steps {
			batchSteps[k] = v
			count++
			if count == config.GetConfig().StepConf.DBUpdateBatch {
				err := SetUsersStepsToCache(batchSteps, background)
				if err != nil {
					return err
				}
				/*
					err = SetUsersStepsToDB(batchSteps, background)
					if err != nil {
						logger.Error("SetUsersStepsToDB error: %v, but redis ok just ignore", err)
					}
				*/
				for _, item := range steps {
					ret, err := pushSetUser(item)
					if err != nil {
						logger.Error("PushSetUser %v err: %v", item.Oid, err)
					} else if ret != 1 {
						logger.Info("PushSetUser %v already exist ", item.Oid)
					} else {
						logger.Info("PushSetUser %v success", item.Oid)
					}
				}
				batchCount += 1
				//logger.Debug("%v - SetUsersSteps batchCount: %v", util.GetCallee(), batchCount)
				count = 0
				batchSteps = make(map[string]*metadata.UserSteps)
			}
		}
		if count > 0 {
			err := SetUsersStepsToCache(batchSteps, background)
			if err != nil {
				return err
			}
			/*
				err = SetUsersStepsToDB(batchSteps, background)
				if err != nil {
					logger.Error("SetUsersStepsToDB error: %v, but redis ok just ignore", err)
				}
			*/
			for _, item := range steps {
				ret, err := pushSetUser(item)
				if err != nil {
					logger.Error("PushSetUser %v err: %v", item.Oid, err)
				} else if ret != 1 {
					logger.Info("PushSetUser %v already exist ", item.Oid)
				} else {
					logger.Info("PushSetUser %v success", item.Oid)
				}
			}
			batchCount += 1
			//logger.Debug("%v - SetUsersSteps batchCount: %v", util.GetCallee(), batchCount)
		}
	} else {
		err := SetUsersStepsToCache(steps, background)
		if err != nil {
			return err
		}
		/*
			err = SetUsersStepsToDB(steps, background)
			if err != nil {
				logger.Error("SetUsersStepsToDB error: %v, but redis ok just ignore", err)
			}
		*/
		for _, item := range steps {
			ret, err := pushSetUser(item)
			if err != nil {
				logger.Error("PushSetUser %v err: %v", item.Oid, err)
			} else if ret != 1 {
				logger.Info("PushSetUser %v already exist ", item.Oid)
			} else {
				logger.Info("PushSetUser %v success", item.Oid)
			}
		}
	}
	return nil
}

// SetUsersStepsToCache ...
func SetUsersStepsToCache(usersSteps map[string]*metadata.UserSteps, background bool) error {
	if len(usersSteps) == 0 {
		return nil
	}
	defer util.GetUsedTime(fmt.Sprintf("%v - count: %v back: %v", util.GetCallee(), len(usersSteps), background))()
	rottenDate, err := util.GetLocalDeltaFormatDate("-720h")
	if err != nil {
		return errors.NewRedisClientError(errors.WithMsg("GetLocalDeltaFormatDate error: %v", err))
	}
	// use pipeline set all user's step into cache
	//pipe := redisClient.Pipeline()
	for user, steps := range usersSteps {
		pipe := redisClient.Pipeline()
		userKey := fmt.Sprintf("%s:%s", STEP_USER_KEY_PREFIX, user)
		var value = make(map[string]interface{})
		for date, step := range steps.Steps {
			value[date] = step
		}
		// add new record
		if len(steps.Steps) >= 30 {
			// rm all data, if offer 30 days step, but h5 only 29 just ignore it
			pipe.Expire(userKey, 0).Result()
		} else {
			// rm old record, if background update
			pipe.HDel(userKey, rottenDate)
		}
		pipe.HMSet(userKey, value)
		// month expire
		pipe.Expire(userKey, time.Hour*24*30).Result()
		// redis exec
		_, err = pipe.Exec()
		if err != nil {
			return errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
		}
		logger.Debug("%v - set user: %v steps count: %v", util.GetCallee(), user, len(steps.Steps))
	}
	return nil
}

// SetUsersStepsToDB ...
func SetUsersStepsToDB(steps map[string]*metadata.UserSteps, background bool) error {
	if len(steps) == 0 {
		return nil
	}
	var users []string
	atomic.AddInt64(&SpeedSetDB, int64(len(steps)))
	defer util.GetUsedTime(fmt.Sprintf("%v - count: %v back: %v", util.GetCallee(), len(steps), background))()
	var autoUpdate = 2
	if background {
		autoUpdate = 1
	}
	sqlStr := "INSERT INTO t_step (`f_user_id`, `f_step`, `f_update_method`, `f_date`, `f_create_time`, `f_modify_time`) VALUES"
	args := make([]interface{}, 0)
	for userID, value := range steps {
		users = append(users, userID)
		for date, count := range value.Steps {
			sqlStr += " (?, ?, ?, ?, now(), now()),"
			args = append(args, userID, count, autoUpdate, date)
		}
	}
	//trim the last
	sqlStr = sqlStr[0 : len(sqlStr)-1]
	// add update
	sqlStr += " on DUPLICATE KEY UPDATE " +
		"f_step = VALUES(f_step)," +
		"f_update_method = VALUES(f_update_method)," +
		"f_modify_time = VALUES(f_modify_time);"
	count, err := cdbClient.ExecSQL(sqlStr, args)
	if err != nil {
		logger.Error("%v - cdbClient.ExecSQL error: %v", util.GetCallee(), err)
		return err
	}
	logger.Debug("%v - users: %v, affect rows: %v", util.GetCallee(), users, count)
	/*
		stmt, err := cdbClient.Prepare(sqlStr)
		if err != nil {
			logger.Error("%v - error: %v", util.GetCallee(), err)
			return err
		}
		defer stmt.(*sql.Stmt).Close()
		res, err := stmt.(*sql.Stmt).Exec(args...)
		if err != nil {
			logger.Error("%v - stmt exec error: %v", util.GetCallee(), err)
			return err
		}
		count, err = res.RowsAffected()
		if err != nil {
			logger.Error("%v - error: %v", util.GetCallee(), err)
			return err
		}
		logger.Debug("%v - affect rows: %v", util.GetCallee(), count)
	*/
	return nil
}
