//
package auto

import (
	"context"
	"database/sql"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/common/wxclient"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/steps"
	"github.com/go-redis/redis"
	"golang.org/x/time/rate"
)

const CHAN_SIZE = 4096
const SET_KEY = "yqz:stepmgr:member"
const HEARTBEAT_INTERVAL = 3

var (
	stopChan    chan struct{}
	cdbClient   dbclient.DBClient
	redisClient *gyredis.RedisClient
	// for auto update
	SpeedPushUpdate int64 = 0
	SpeedUpdate     int64 = 0
	limiter         *rate.Limiter
)

func InitAutoUpdateComponent(stepConf config.StepConfig, cdb dbclient.DBClient, redisCli *gyredis.RedisClient) error {
	logger.Sys("Try InitAutoUpdateComponent...")
	if !config.GetConfig().StepConf.Flag {
		logger.Sysf("auto update flag is false, do not start auto update")
		return nil
	}
	stopChan = make(chan struct{})
	dataChan := make(chan *wxclient.UserInfo, CHAN_SIZE)
	cdbClient = cdb
	redisClient = redisCli
	// init limiter
	limiter = rate.NewLimiter(rate.Every(100*time.Millisecond), config.GetConfig().StepConf.AutoLimit)
	// init kafka sender
	err := KafkaSenderInit()
	if err != nil {
		logger.Sysf("KafkaSenderInit error: %v", err)
	}
	// init yqz auto func
	go runAutoYQZFunc(stepConf.UpdateIntervalS)
	// run heartbeat
	go heartbeatWorker()
	// run dispatcher worker
	go initDispatcher(stepConf.UpdateIntervalS, dataChan)
	// run auto update worker
	for i := 0; i < stepConf.UpdateCount; i++ {
		go initAutoUpdateWorker(i, dataChan)
	}
	return nil
}

func CloseAutoUpdateComponent() {
	logger.Sys("Try CloseAutoUpdateComponent...")
	if !config.GetConfig().StepConf.Flag {
		logger.Sysf("auto update flag is false, do not start auto update")
		return
	}
	close(stopChan)
	time.Sleep(1 * time.Second)
}

func initDispatcher(interval int, dataChan chan<- *wxclient.UserInfo) {
	// run dispatcher worker
	logger.Sys("DispatcherWorker is start")
	var lastTotal, lastIndex int
	t := time.NewTimer(time.Duration(interval) * time.Second)
	for {
		select {
		case <-stopChan:
			logger.Sys("DispatcherWorker is stop")
			t.Stop()
			return
		case <-t.C:
			logger.Sysf("AutoDispatcherWorker run time: %v", util.GetLocalFormatTime())
			// find out handle range for this process
			totalIns, indexIns, err := getWorkerCountAndIndex()
			if err != nil {
				logger.Error("get all instance info err: %v, use old result total: %v, index: %v", err, lastTotal, lastIndex)
			} else {
				if indexIns == -1 {
					logger.Errorf("can not get instance: %v index", util.GetLocalIP())
					t.Reset(time.Duration(HEARTBEAT_INTERVAL) * time.Second)
					continue
				}
				lastTotal, lastIndex = totalIns, indexIns
			}
			// get all user
			wg := &sync.WaitGroup{}
			ctx, cancel := context.WithTimeout(context.TODO(), time.Second*time.Duration(interval))
			defer cancel()
			var total int32
			sqlStr := "select f_user_id, f_unin_id, f_appid from t_user where f_backend_update = 2 limit ?, ?"
			for i := 0; ; i++ {
				// collect batch user from db
				var result []*wxclient.UserInfo
				offset := indexIns*config.GetConfig().StepConf.DBFetchBatch + totalIns*(i)*config.GetConfig().StepConf.DBFetchBatch
				args := []interface{}{offset, config.GetConfig().StepConf.DBFetchBatch}
				res, err := cdbClient.Query(sqlStr, args)
				if err != nil {
					logger.Errorf("QueryDB error: %v", err)
					continue
				}
				for _, item := range res {
					info := &wxclient.UserInfo{
						UserID: item["f_user_id"],
						UninID: item["f_unin_id"],
						Appid:  item["f_appid"],
					}
					result = append(result, info)
				}
				// send all result to dataChan
				wg.Add(1)
				go func(ctx context.Context, result []*wxclient.UserInfo) {
					var part int32
				loop:
					for _, value := range result {
						select {
						case dataChan <- value:
							part++
						case <-ctx.Done():
							logger.Error("timeout, %v second can not finish auto update step, may need increase handle thread", time.Duration(interval))
							break loop
						}
					}
					atomic.AddInt32(&total, part)
					atomic.AddInt64(&SpeedPushUpdate, int64(part))
					wg.Done()
				}(ctx, result)
				// if finish fetch all user from db
				if len(result) != config.GetConfig().StepConf.DBFetchBatch {
					break
				}
			}
			// wait all thread finish util timeout or finish
			wg.Wait()
			logger.Sysf("AutoDispatcherWorker dispatch total user count: %v", total)
			t.Reset(time.Duration(interval) * time.Second)
		}
	}
}

func getWorkerCountAndIndex() (int, int, error) {
	startTime := util.GetLocalTime().Add(-1 * time.Minute).Unix()
	opt := redis.ZRangeBy{
		Min: strconv.Itoa(int(startTime)),
		Max: strconv.Itoa(int(util.GetLocalTime().Unix())),
	}
	res, err := cacheclient.RedisClient.ZRangeByScore(SET_KEY, opt).Result()
	if err != nil {
		logger.Error("ZRange client exec error, err = %v", err)
		return 0, 0, err
	}
	sort.Strings(res)
	var index int = -1
	for i, v := range res {
		if v == util.GetLocalIP() {
			index = i
		}
	}
	logger.Debug("get range from: [%v - %v], res: %v, index: %v", opt.Min, opt.Max, len(res), index)
	return len(res), index, nil
}

func heartbeatWorker() {
	logger.Sys("AutoDispatcherWorker heartbeatWorker is start")
	for {
		select {
		case <-stopChan:
			logger.Sys("AutoDispatcherWorker heartbeatWorker is stop")
			_, err := cacheclient.RedisClient.ZRem(SET_KEY, util.GetLocalIP()).Result()
			if err != nil {
				logger.Error("remove instance: %v in set error: %v", util.GetLocalIP(), err)
			}
			return
		default:
			timestamp := time.Now().Unix()
			cacheclient.RedisClient.ZAdd(SET_KEY, redis.Z{
				Score:  float64(timestamp),
				Member: util.GetLocalIP(),
			})
			//logger.Sys("AutoDispatcherWorker %v heartbeat time: %v", util.GetLocalIP(), util.GetLocalFormatTime())
			time.Sleep(HEARTBEAT_INTERVAL * time.Second)
		}
	}
}

func initAutoUpdateWorker(index int, dataChan <-chan *wxclient.UserInfo) {
	// run update worker
	logger.Sys("AutoUpdateWorker[%v] is start", index)
	var users []*wxclient.UserInfo
	// handle per 100ms
	timer := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-stopChan:
			logger.Sys("AutoUpdateWorker[%v] is stop", index)
			return
		case user := <-dataChan:
			// handle auto update
			users = append(users, user)
			if len(users) >= config.GetConfig().StepConf.DBUpdateBatch {
				// handle users
				logger.Debug("auto update user reach limit: [%v]", len(users))
				go UpdateUserSteps(users)
				users = make([]*wxclient.UserInfo, 0)
			}
		case <-timer.C:
			if len(users) != 0 {
				// handle users
				logger.Debug("auto update reach interval count: [%v]", len(users))
				go UpdateUserSteps(users)
				users = make([]*wxclient.UserInfo, 0)
			}
		}
	}
}

// Update all users steps
func UpdateUserSteps(users []*wxclient.UserInfo) error {
	var wgList sync.WaitGroup = sync.WaitGroup{}
	var muList sync.Mutex = sync.Mutex{}
	var userStepList []wxclient.WxStepsData
	// check wx fetch batch
	batch := len(users) / config.GetConfig().StepConf.WxUpdateBatch
	if len(users)%config.GetConfig().StepConf.WxUpdateBatch != 0 {
		batch += 1
	}
	for i := 0; i < batch; i++ {
		// get steps from wx
		begin := i * config.GetConfig().StepConf.WxUpdateBatch
		if begin >= len(users) {
			break
		}
		end := (i + 1) * config.GetConfig().StepConf.WxUpdateBatch
		if end > len(users) {
			end = len(users)
		}
		wgList.Add(1)
		go func() {
			stepRes, err := wxclient.GetUsersStepFromWX(users[begin:end])
			if err != nil {
				logger.Errorf("wxclient.GetUsersStepFromWX error: %v", err)
			} else {
				muList.Lock()
				userStepList = append(userStepList, stepRes.UserStepList...)
				muList.Unlock()
			}
			wgList.Done()
		}()
	}
	wgList.Wait()
	// here we need control write db limit, 1000 * 10 / s
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	err := limiter.Wait(ctx)
	if err != nil {
		logger.Error("wait rate limit token timeout, loss batch user count: %v", len(userStepList))
	}

	// set steps to cache and db
	rottenUsers, err := handleUsersStep(users, userStepList)
	if err != nil {
		return err
	}
	// handle rotten user
	err = handleRottenUsers(rottenUsers)
	if err != nil {
		return err
	}
	atomic.AddInt64(&SpeedUpdate, int64(len(users)))
	logger.Sysf("AutoUpdateWorker update step user count: %v", len(users))
	return nil
}

//func handleUsersStep(users []*wxclient.UserInfo, wxStepsResult *wxclient.WxStepsRes) ([]string, error) {
func handleUsersStep(users []*wxclient.UserInfo, userStepList []wxclient.WxStepsData) ([]string, error) {
	defer util.GetUsedTime("handleUsersStep")()
	if len(userStepList) == 0 {
		logger.Infof("get no user step info")
		return nil, nil
	}
	// handle user info
	var uninOid = make(map[string]string)
	var oidUnin = make(map[string]string)
	for _, value := range users {
		uninOid[value.UninID] = value.UserID
		oidUnin[value.UserID] = value.UninID
	}
	// handle steps response
	rottenUsers := make([]string, 0)
	usersSteps := make(map[string]*metadata.UserSteps)
	for _, it := range userStepList {
		logger.Debug("oid = %s, wx ret = %+v", uninOid[it.Openid], it)
		if _, ok := uninOid[it.Openid]; !ok {
			logger.Error("can not get user unin: %v relative oid", it.Openid)
			continue
		}
		userID := uninOid[it.Openid]
		if it.Errcode == 20070 || it.Errcode == 20071 {
			rottenUsers = append(rottenUsers, userID)
			continue
		}
		date_steps := make(map[string]int64)
		userSteps := &metadata.UserSteps{Oid: userID, Steps: make(map[string]int32)}
		for _, value := range it.StepList {
			tm := time.Unix(value.Timestamp, 0)
			date := tm.Format("2006-01-02")
			date_steps[date] = value.Step
			userSteps.Steps[date] = int32(value.Step)
		}
		usersSteps[userID] = userSteps
	}
	// set cache & db
	err := steps.SetUsersSteps(usersSteps, true)
	if err != nil {
		logger.Error("steps.SetUsersSteps error: %v", err)
		return nil, err
	}
	// send step info to kafka for other consume
	if err := SendSteps2KafkaLocal(usersSteps, oidUnin); err != nil {
		logger.Error("send step to kafka error: %v", err)
	}
	// handle yqz state
	if err := HandleYQZState(usersSteps); err != nil {
		logger.Error("HandleYQZState error: %v", err)
	}
	logger.Info("updateStep total user count: %v", len(usersSteps))
	return rottenUsers, nil
}

func handleRottenUsers(rottenUsers []string) error {
	defer util.GetUsedTime("handleRottenUsers")()
	if len(rottenUsers) == 0 {
		logger.Infof("get no rotten user info")
		return nil
	}
	rawSqlStr := "UPDATE t_user SET f_backend_update = 1, f_modify_time = now() WHERE f_user_id in ("
	// handle last index
	batchCnt := len(rottenUsers) / config.GetConfig().StepConf.DBUpdateBatch
	sqlStr := rawSqlStr
	args := make([]interface{}, 0)
	for _, value := range rottenUsers[batchCnt*config.GetConfig().StepConf.DBUpdateBatch:] {
		sqlStr += "?,"
		args = append(args, value)
	}
	sqlStr = sqlStr[0 : len(sqlStr)-1]
	sqlStr += ")"
	stmt, err := cdbClient.Prepare(sqlStr)
	if err != nil {
		logger.Error("cdbClient.Prepare error: %v", err)
		return err
	}
	defer stmt.(*sql.Stmt).Close()
	res, err := stmt.(*sql.Stmt).Exec(args...)
	if err != nil {
		logger.Error("stmt exec error: %v", err)
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		logger.Error("cdbClient.Prepare error: %v", err)
		return err
	}
	logger.Debug("affect rows: %v", count)
	// handle other batch update
	if len(rottenUsers) >= config.GetConfig().StepConf.DBUpdateBatch {
		sqlStr = rawSqlStr
		for i := 0; i < config.GetConfig().StepConf.DBUpdateBatch; i++ {
			sqlStr += "?,"
		}
		sqlStr = sqlStr[0 : len(sqlStr)-1]
		sqlStr += ")"
		stmtFull, err := cdbClient.Prepare(sqlStr)
		if err != nil {
			logger.Error("cdbClient.Prepare error: %v", err)
			return err
		}
		defer stmtFull.(*sql.Stmt).Close()

		for _, value := range rottenUsers[:batchCnt*config.GetConfig().StepConf.DBUpdateBatch] {
			args = append(args, value)
			if len(args) >= config.GetConfig().StepConf.DBUpdateBatch {
				res, err := stmt.(*sql.Stmt).Exec(args...)
				if err != nil {
					logger.Error("stmt exec error: %v", err)
					return err
				}
				count, err := res.RowsAffected()
				if err != nil {
					logger.Error("res.RowsAffected error: %v", err)
					return err
				}
				logger.Debug("affect rows: %v", util.GetCallee(), count)
			}
		}
	}
	logger.Info("handle total rotten user count: %v", len(rottenUsers))
	return nil
}
