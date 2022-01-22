package event_certificate_msg

import (
	"fmt"
	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/business_access"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"github.com/go-redis/redis"
	"time"
)

const (
	GLOBAL_WEEK_Certificate_HASH_KEY = "yqz_week_certificate_hash_%s"
)

func CertificateTask() {
	return
	for {
		// 本周的证书
		now := time.Now()
		err := certificateTask(now)
		if err != nil {
			log.Error("t = %v, certificateTask error, err = %v", now, err)
		}
		// 上周的证书
		lastWeek := now.AddDate(0, 0, -7)
		err = certificateTask(lastWeek)
		if err != nil {
			log.Error("t = %v, certificateTask error, err = %v", lastWeek, err)
		}

		time.Sleep(time.Minute * 10)
	}
}

func certificateTask(t time.Time) error {
	db_proxy := data_access.DBProxy{
		DBHandler: data_access.DBHandler,
		Tx:        false,
	}

	startTime, endTime := common.WeekStartEnd(t)
	weekID := startTime.Format("2006-01-02")
	week := weekByCurYear(startTime)

	// 地图信息
	opl, err := getWeekMap(startTime)
	if err != nil {
		log.Error("getWeekMap error, err = %v", err)
		return err
	}
	mapTitle := opl.Name
	mapName := opl.Abbr
	dis := opl.Distance

	// 大地图总人数
	total, err := totalUserBigMap(weekID)
	if err != nil {
		log.Error("certificate error, err = %v", err)
		return err
	}

	// 初始化是否已经写入过证书
	err = initCertificate(weekID, endTime)
	if err != nil {
		log.Error("initCertificate error, err = %v", err)
		return err
	}

	// 批量生成证书
	var offset, count int64 = 0, 999
	for offset < total {
		list, err := rangeUserBigMap(weekID, offset, offset+count)
		if err != nil {
			log.Error("rangeUserBigMap error, err = %v", err)
			continue
		}

		for i, v := range list {
			oid := v.Member.(string)
			step := int64(v.Score)

			// 判断是否到达终点
			proc := 0
			if opl.Distance > 0 {
				proc = int(float64(step)*common.StepToMI) / opl.Distance
			}
			if proc == 0 {
				log.Info("not round 1, oid:%v, step:%v", oid, step)
				continue
			}

			// 是否已经写入过证书
			isCert, err := isCertificate(weekID, oid)
			if err != nil {
				log.Error("isCertificate error, err = %v", err)
				continue
			}
			if isCert == true {
				continue
			}

			log.Debug("certificate weekID:%v, oid:%v, week:%v, mapTitle:%v, mapName:%v, dis:%v, userNum:%v, rank:%v", weekID, oid, week, mapTitle, mapName, dis, total, offset+int64(i)+1)
			// 生成证书
			content, err := certificate(week, mapTitle, mapName, dis, total, offset+int64(i)+1)
			if err != nil {
				log.Error("certificate error, err = %v", err)
				continue
			}
			ch := data_access.CertificateHistory{
				FWeek:       weekID,
				FUserId:     oid,
				FContent:    content,
				FStatus:     1,
				FCreateTime: time.Now().Format("2006-01-02 15:04:05"),
				FModifyTime: time.Now().Format("2006-01-02 15:04:05"),
			}
			err = business_access.InsertCertificateHistory(db_proxy, ch)
			if err != nil {
				log.Error("InsertCertificateHistory error, err = %v, ch = %v", err, ch)
				continue
			}

			// 写入证书记录
			err = setCertificate(weekID, oid)
			if err != nil {
				log.Error("setCertificate error, err = %v", err)
				continue
			}
		}

		offset += count + 1
	}

	return nil
}

func totalUserBigMap(weekID string) (int64, error) {
	key := fmt.Sprintf("yqz_gb_%s_week_rank", weekID)

	total, err := cache_access.RedisClient.ZCard(key).Result()
	if err != nil {
		log.Error("key = %s, redis ZCard err = %v", key, err)
		return 0, err
	}

	return total, nil
}

func rangeUserBigMap(weekID string, start, stop int64) ([]redis.Z, error) {
	key := fmt.Sprintf("yqz_gb_%s_week_rank", weekID)

	results, err := cache_access.RedisClient.ZRevRangeWithScores(key, start, stop).Result()
	if err != nil {
		log.Error("key = %s, start = %d, stop = %d redis ZRevRangeWithScores err = %v", key, start, stop, err)
		return nil, err
	}

	return results, nil
}

func initCertificate(weekID string, endTime time.Time) error {
	key := fmt.Sprintf(GLOBAL_WEEK_Certificate_HASH_KEY, weekID)
	exist, err := cache_access.RedisClient.Exists(key).Result()
	if err != nil {
		log.Error("redis Exists, err = %v, key = %v", err, key)
		return err
	}

	if exist == 1 {
		return nil
	}

	pl := cache_access.RedisClient.Pipeline()

	pl.HSet(key, "init", "1")

	diff := endTime.Unix() - time.Now().Unix()
	pl.Expire(key, time.Duration(diff)*time.Second)

	cmd, err := pl.Exec()
	if err != nil {
		log.Error("redis pipeline error, err = %v, cmd = %v", err, cmd)
		return err
	}

	return nil
}

func setCertificate(weekID, oid string) error {
	key := fmt.Sprintf(GLOBAL_WEEK_Certificate_HASH_KEY, weekID)
	_, err := cache_access.RedisClient.HSet(key, oid, 1).Result()
	if err != nil {
		log.Error("redis HSet error, err = %v", err)
		return err
	}
	return nil
}

func isCertificate(weekID, oid string) (bool, error) {
	key := fmt.Sprintf(GLOBAL_WEEK_Certificate_HASH_KEY, weekID)
	_, err := cache_access.RedisClient.HGet(key, oid).Result()
	if err == redis.Nil {
		log.Info("redis HGet key = %s, not exist", key)
		return false, nil
	}

	if err != nil && err != redis.Nil {
		log.Error("redis client HGet error, err = %v, key = %s", err, key)
		return false, err
	}

	return true, nil
}

func certificate(week int, mapTitle, mapName string, dis int, userNum, rank int64) (string, error) {
	tn := time.Now()
	cd := common.GetCertificateV2Res{
		Week:             week,
		MapTitle:         mapTitle,
		MapName:          mapName,
		MapDistance:      dis / 1000,
		MapDistanceMeter: dis,
		Exceed:           userNum - rank,
		FinishTime:       tn.Format("2006-01-02 15:04:05"),
		Date:             tn.Format("2006-01-02"),
	}

	js, err := json.Marshal(cd)
	if err != nil {
		log.Error("Marshal, err = %v, cd = %v", err, cd)
		return "", err
	}

	return string(js), nil
}

//判断时间是当年的第几周
func weekByCurYear(t time.Time) int {
	yearDay := t.YearDay()
	yearFirstDay := t.AddDate(0, 0, -yearDay)
	firstDayInWeek := int(yearFirstDay.Weekday())

	//今年第一周有几天
	firstWeekDays := 1
	if firstDayInWeek != 0 {
		firstWeekDays = 7 - firstDayInWeek + 1
	}
	var week int
	if yearDay <= firstWeekDays {
		week = 1
	} else {
		week = (yearDay-firstWeekDays)/7 + 2
	}
	return week
}
