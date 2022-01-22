//
package auto

import (
	"fmt"
	"sync"
	"time"

	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/user"
)

var yqzUser sync.Map

// for PK ...
const PK_HASH_KEY = "yqz_pk_hash_key"
const PK_HASH_EXPIRE = 60 * 60 * 24 * 15

type PKStep struct {
	Date string `json:"date"`
	Step int32  `json:"step"`
}

func runAutoYQZFunc(interval int) error {
	// get data when start
	logger.Sysf("AutoUpdateYQZUsers run time: %v", util.GetLocalFormatTime())
	err := autoUpdateYQZUsers()
	if err != nil {
		logger.Error("autoUpdateYQZUsers error: %v", err)
	}
	// start auto update worker
	go func() {
		logger.Sys("AutoYQZWorker is start")
		t := time.NewTimer(time.Duration(interval) * time.Second)
		for {
			select {
			case <-stopChan:
				logger.Sys("AutoYQZWorker is stop")
				t.Stop()
				return
			case <-t.C:
				logger.Sysf("AutoUpdateYQZUsers run time: %v", util.GetLocalFormatTime())
				err := autoUpdateYQZUsers()
				if err != nil {
					logger.Error("AutoUpdateYQZUsers error: %v", err)
				}
			}
		}
	}()
	return nil
}

// checkUserInYQZ check if user is in yqz activity
func checkUserInYQZ(oid string) (bool, error) {
	/*
		_, ok := yqzUser.Load(oid)
		return ok
	*/
	profile, err := user.GetUserProfile(redisClient, oid)
	if err != nil {
		logger.Error("user.GetUserProfile oid: %v, error: %v", oid, err)
		return false, err
	}
	if profile.Donated == 1 {
		return true, nil
	}
	return false, nil
}

// checkUserInCurrentYQZ check if user is in current yqz activity
func checkUserInCurrentYQZ(oid string) bool {
	_, ok := yqzUser.Load(oid)
	return ok
}

// yQZUserAdd add user in yqz activity
func yQZUserAdd(oid string) {
	yqzUser.Store(oid, true)
}

// autoUpdateYQZUsers check if user is in yqz activity
func autoUpdateYQZUsers() error {
	currentTime := util.GetLocalTime()
	start, _ := util.WeekStartEnd(currentTime)
	startDate := start.Format("2006-01-02")
	sqlStr := "select DISTINCT(f_user_id) as user_id from t_global_step where f_period = ? limit ?, ?"
	var total = 0
	for i := 0; ; i++ {
		// collect batch user from db
		offset := i * config.GetConfig().StepConf.DBFetchBatch
		args := []interface{}{startDate, offset, config.GetConfig().StepConf.DBFetchBatch}
		res, err := cdbClient.Query(sqlStr, args)
		if err != nil {
			logger.Errorf("QueryDB error: %v", err)
			return err
		}
		for _, item := range res {
			yqzUser.Store(item["user_id"], true)
		}
		// if finish fetch all user from db
		total += len(res)
		if len(res) != config.GetConfig().StepConf.DBFetchBatch {
			break
		}
	}
	logger.Sys("autoUpdateYQZUsers time: %v get period: %v total yqz user count: %v", util.GetLocalFormatTime(), startDate, total)
	return nil
}

// HandleYQZState ...
func HandleYQZState(userStep map[string]*metadata.UserSteps) error {
	var inUser = make(map[string]*metadata.UserSteps)
	var outUser = make(map[string]*metadata.UserSteps)
	for oid, steps := range userStep {
		exist, _ := checkUserInYQZ(oid)
		if exist {
			inUser[oid] = steps
		} else {
			outUser[oid] = steps
		}
	}
	logger.Infof("handle global in user: %v , out user: %v", len(inUser), len(outUser))
	// update pk step, here is not rational, it will remove in future
	err := updatePKStep(userStep)
	if err != nil {
		logger.Error("updatePKStep error: %v", err)
		return err
	}
	// handle in yqz users
	err = handleInYQZUser(inUser)
	if err != nil {
		logger.Error("handleInYQZUser error: %v", err)
		return err
	}
	return nil
}

func handleInYQZUser(userStep map[string]*metadata.UserSteps) error {

	return nil
}

func updatePKStep(usersStep map[string]*metadata.UserSteps) error {
	start, _ := util.WeekStartEnd(util.GetLocalTime())
	startDate := start.Format("2006-01-02")
	currentDate := util.GetLocalFormatDate()
	for oid, userStep := range usersStep {
		var pkSteps []PKStep
		for ; ; start = start.AddDate(0, 0, 1) {
			startDate := start.Format("2006-01-02")
			// 当天步数不同步
			if startDate >= currentDate {
				break
			}
			if s, ok := userStep.Steps[startDate]; ok {
				pkSteps = append(pkSteps, PKStep{
					Date: startDate,
					Step: s,
				})
			}
		}
		if len(pkSteps) > 0 {
			setPKStepsCache(startDate, oid, pkSteps)
		}
	}
	return nil
}

func setPKStepsCache(week, oid string, steps []PKStep) error {
	key := fmt.Sprintf("%s_%s_%s", PK_HASH_KEY, week, oid)
	pipe := cacheclient.RedisClient.Pipeline()
	for _, v := range steps {
		pipe.HSet(key, v.Date, v.Step)
	}
	pipe.Expire(key, PK_HASH_EXPIRE*time.Second)
	cmd, err := pipe.Exec()
	if err != nil {
		logger.Error("redis pipeline error, err = %v, cmd = %+v", err, cmd)
		return err
	}
	return nil
}
