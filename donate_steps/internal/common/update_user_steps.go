//
//

package common

import (
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/internal/business_access"
	"git.code.oa.com/gongyi/donate_steps/pkg/stepmgr"
)

func UpdateEventInfo(oid string, date_steps map[string]int64, auto bool) error {
	events, err := business_access.GetEventUserbyUidStatus(oid, 2)
	if err != nil {
		log.Error("GetEventUserbyUidStatus error, err= %v, oid = %s", err, oid)
		return err
	}
	tmp_date_steps := make(map[string]int64)
	for k, v := range date_steps {
		tmp_date_steps[k] = v
	}
	done := false
	nt := time.Now()
	for _, it := range events {
		event_info, err := business_access.GetEventInfoByEid(it)
		if err != nil {
			log.Error("GetEventInfoByEid error, err = %v, event_id = %s", err, it)
			return err
		}
		if event_info.FRouteId == 0 || event_info.FStatus != 2 {
			log.Info("event info = %+v, route id = 0 or status != 2", event_info)
			continue
		}

		if len(event_info.FEventId) != 0 && len(date_steps) != 0 {
			tx, err := data_access.DBHandler.Begin()
			if err != nil {
				log.Error("begin db transaction error, err = %s", err.Error())
				return err
			}
			defer tx.Rollback()
			db_proxy := data_access.DBProxy{
				DBHandler: tx,
				Tx:        true,
			}

			err = DonateStepsV2(db_proxy, it, oid, nt, date_steps)
			if err != nil {
				log.Error("DonateStepsV2 error, err = %v, eid = %s, oid = %s, date_steps = %+v", err, it, oid, date_steps)
				return err
			}
			log.Info("new eid = %s, oid = %s", it, oid)

			if !done {
				err = InitiativeUpdateSteps(db_proxy, oid, tmp_date_steps, auto)
				if err != nil {
					log.Error("InitiativeUpdateSteps error, err = %v", err)
					return err
				}
				done = true
			}

			err = tx.Commit()
			if err != nil {
				log.Error("tx commit error, err = %v", err)
				return err
			}
		}
	}
	if len(events) == 0 {
		// 兼容新版本大地图
		//nt := time.Now()
		//start, end := WeekStartEnd(nt)
		//start_date := start.Format("2006-01-02")
		//end_date := end.Format("2006-01-02")
		//
		//sum, err := business_access.GetSumStepsByDates(oid, start_date, end_date)
		//if err != nil {
		//	log.Error("GetSumStepsByDates error, err = %v, oid = %s, sd = %s, ed = %s", err, oid, start_date, end_date)
		//	return err
		//}

		//_, err = BigMapDonateStepV2(oid, start_date, end_date, nt, sum, date_steps)
		//if err != nil {
		//	log.Error("BigMapDonateStepV2 error, err = %v, oid = %s, date_steps = %+v", err, oid, date_steps)
		//	return err
		//}
	}
	return nil
}

// InitiativeUpdateSteps update step for db & redis
func InitiativeUpdateSteps(db_proxy data_access.DBProxy, oid string, dateSteps map[string]int64, auto bool) error {
	// call stepmgr update cache and db
	rsp, err := stepmgr.SetUserSteps(oid, dateSteps, auto)
	if err != nil {
		log.Error("call stepmgr SetUserSteps error, err = %v", err)
		return err
	} else {
		if rsp.Header.Code != 0 {
			log.Error("SetUserSteps logical error, err = %v", err)
			return err
		}
	}

	// todo, remove pk step in future
	// 本周有回补数据，更新pk step
	week := GetFirstDateOfWeek()
	t, _ := time.ParseInLocation("2006-01-02", week, Loc)
	nt := time.Now().Format("2006-01-02")
	var pkSteps []cache_access.PKStep
	for ; ; t = t.AddDate(0, 0, 1) {
		tf := t.Format("2006-01-02")
		// 当天步数不同步
		if tf >= nt {
			break
		}
		if s, ok := dateSteps[tf]; ok {
			pkSteps = append(pkSteps, cache_access.PKStep{
				Date: tf,
				Step: s,
			})
		}
	}
	if len(pkSteps) > 0 {
		cache_access.SetPKSteps(week, oid, pkSteps)
	}
	return nil
}
