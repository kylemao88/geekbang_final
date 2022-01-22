//

package statistic

import (
	"strconv"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/pkg/dbclient"
)

var (
	dbClient dbclient.DBClient
	stopChan chan struct{}
	dbConfig dbclient.DBConfig
)

func InitialStatistic(cfg dbclient.DBConfig) error {
	var err error
	dbConfig = cfg
	dbClient, err = dbclient.GetDBClientByCfg(dbConfig)
	if err != nil {
		log.Error("InitialStatistic error: %v", err)
		return err
	}
	stopChan = make(chan struct{})
	go RunDailyStatistic()
	return nil
}

func Close() {
	dbclient.CloseClient(dbConfig.DBType, dbClient)
	close(stopChan)
}

func RunDailyStatistic() {
	log.Info("RunDailyStatistic is start...")
	// daily job
	now := time.Now()
	next := now.Add(time.Hour * 24)
	next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	pkDAUTimer := time.NewTimer(next.Sub(now))
	// weekly job
	firstDate := getFirstDateOfWeek()
	nextWeek := firstDate.Add(time.Hour * 24 * 7)
	nextWeek = time.Date(nextWeek.Year(), nextWeek.Month(), nextWeek.Day(), 0, 0, 0, 0, nextWeek.Location())
	pkWAUTimer := time.NewTimer(nextWeek.Sub(now))
	for {
		select {
		case <-stopChan:
			log.Info("RunDailyStatistic is stop...")
			return
		case <-pkDAUTimer.C:
			log.Info("newPKUserDAU is trigger..., time: %v", time.Now().Format("2006-01-02 15:04:05"))
			now = time.Now()
			next = now.Add(time.Hour * 24)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			pkDAUTimer = time.NewTimer(next.Sub(now))
			go newPKUserDAU()
		case <-pkWAUTimer.C:
			log.Info("newPKUserWAU is trigger..., time: %v", time.Now().Format("2006-01-02"))
			firstDate = getFirstDateOfWeek()
			nextWeek = firstDate.Add(time.Hour * 24 * 7)
			nextWeek = time.Date(nextWeek.Year(), nextWeek.Month(), nextWeek.Day(), 0, 0, 0, 0, nextWeek.Location())
			pkWAUTimer = time.NewTimer(nextWeek.Sub(now))
			go newPKUserWAU()
		default:
			// loop interval
			time.Sleep(time.Duration(60) * time.Second)
		}
	}
}

func newPKUserDAU() {
	user_sql_str := "select f_user_id from t_user_source where f_type_id = 1"
	sql_str := "select f_user_id, COUNT(*) as total from t_user_behavior " +
		"where f_create_time > ? and f_type_id = 0 and f_user_id in (" + user_sql_str + ") " +
		"group by f_user_id"
	start := time.Now().AddDate(0, 0, -1).Format("2006-01-02 15:04:05")
	args := []interface{}{start}
	rows, err := dbClient.Query(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return
	}
	for _, item := range rows {
		total, err := strconv.Atoi(item["total"])
		if err != nil {
			log.Error("Unexpect error: $v", err)
			continue
		}
		err = InsertUserDAU(item["f_user_id"], total)
		if err != nil {
			log.Error("InsertUserWAU error: $v", err)
		}
	}
}

// pk user visit yqz times per week
func newPKUserWAU() {
	user_sql_str := "select f_user_id from t_user_source where f_type_id = 1"
	sql_str := "select f_user_id, COUNT(*) as total from t_user_behavior " +
		"where f_create_time > ? and f_type_id = 0 and f_user_id in (" + user_sql_str + ") " +
		"group by f_user_id"
	start := getFirstDateOfLastWeek().Format("2006-01-02 15:04:05")
	args := []interface{}{start}
	rows, err := dbClient.Query(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return
	}
	for _, item := range rows {
		total, err := strconv.Atoi(item["total"])
		if err != nil {
			log.Error("Unexpect error: $v", err)
			continue
		}
		err = InsertUserWAU(item["f_user_id"], total)
		if err != nil {
			log.Error("InsertUserWAU error: $v", err)
		}
	}
}

func getFirstDateOfWeek() (weekStartDate time.Time) {
	now := time.Now()

	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	weekStartDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	return
}

func getFirstDateOfLastWeek() (lastWeekMonday time.Time) {
	thisWeekMonday := getFirstDateOfWeek()
	lastWeekMonday = thisWeekMonday.AddDate(0, 0, -7)
	return
}
