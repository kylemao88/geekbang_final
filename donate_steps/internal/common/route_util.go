//
//

package common

import (
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/internal/business_access"
	"time"
)

const StepToMI = 0.7

type RouteCheckStatus struct {
	Status     int
	ArriveDate string
}

var LevelNameToCityName = map[string][]string{
	"冬奥冰雪之旅":  {"北京", "张家口"},
	"孔孟故里之旅":  {"济南", "曲阜"},
	"徽派古村落之旅": {"黄山"},
	"江南水乡之旅":  {"苏州"},
	"玄奘之路戈壁线": {"吐鲁番"},
	"冈仁波齐转山":  {"阿里"},
	"青海湖环线":   {"西宁"},
	"海南环岛线":   {"海口", "三亚"},
	"京城畅游线":   {"北京"},
	"上海环江线":   {"上海"},
	"杭州西湖环线":  {"杭州"},
	"西安环城线":   {"西安"},
	"广州游玩线":   {"广州"},
	"武汉绕城环线":  {"武汉"},
}

var WeekNameToCityName = map[string][]string{
	"黄山之行":    {"黄山"},
	"华山之行":    {"西安"},
	"玄武湖钟山之行": {"南京"},
	"西湖之行":    {"杭州"},
}

func InsertRouteInfoToDB(db_proxy data_access.DBProxy, eid string, rid int) error {
	_, route_data_map, err := GetRouteLevelMapAndRouteMap(rid)
	if err != nil {
		log.Error("GetRouteDataMap error, err = %v", err)
		return err
	}

	if _, ok := route_data_map[rid]; ok {
		route_info_list := route_data_map[rid].LevelList
		level_size := 0
		for _, it := range route_info_list {
			if it.Status {
				level_size++
			}
		}

		max_point, _, err := business_access.GetRouteStatMaxInsertPoint(eid, rid)
		if err != nil {
			log.Error("GetRouteStatMaxInsertPoint error, err = %v, eid = %s, rid = %d", err, eid, rid)
			return err
		}

		nt_date := time.Now().Format("2006-01-02 15:04:05")
		if max_point < level_size {
			orid := max_point
			for _, it := range route_info_list {
				if it.Status {
					info, err := data_access.TxGetRouteStatByLevelId(db_proxy, eid, rid, it.Did)
					if err != nil {
						log.Error("GetRouteStat error, err = %v, eid = %s, rid = %d", err, eid, rid)
						return err
					}
					if len(info.FEventId) == 0 {
						orid = orid + 1
						route_stat := data_access.RouteStat{
							FEventId:    eid,
							FRouteId:    rid,
							FOrderId:    orid,
							FLevelId:    it.Did,
							FFinishTime: nt_date,
							FStatus:     0,
							FCreateTime: nt_date,
							FModifyTime: nt_date,
						}
						if err = business_access.InsertRouteStat(db_proxy, route_stat); err != nil {
							log.Error("InsertRouteStat error, err = %v, route_stat = %v", err, route_stat)
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func AddEventRouteStat(db_proxy data_access.DBProxy, eid, nt_date string, rid, orid int) error {
	oids, err := business_access.GetEventUserID(eid)
	if err != nil {
		log.Error("GetEventUserID error, err = %v", err)
		return err
	}

	norid := orid + 1
	for _, id := range oids {
		exist, err := business_access.IsInEventRouteStat(eid, id, norid)
		if err != nil {
			log.Error("IsInEventRouteStat error, err = %v, eid = %s, oid = %s, current_point = %d",
				err, eid, id, orid)
			return err
		}
		if !exist {
			ers := data_access.EventRouteStat{
				FEventId:      eid,
				FRouteId:      rid,
				FOrderId:      norid,
				FUserId:       id,
				FFinishTIme:   nt_date,
				FInTeamStatus: 2,
				FCreateTime:   nt_date,
				FModifyTime:   nt_date,
			}
			if err = business_access.InsertEventRouteStat(db_proxy, ers); err != nil {
				log.Error("InsertEventRouteStat error, err = %v, oid = %v", err, id)
				return err
			}
		}
	}
	return nil
}

func FinishRoute(db_proxy data_access.DBProxy, eid, nt_date string, rid, orid int, route_map map[int]RouteInfo) error {
	if rt, ok := route_map[rid]; ok {
		route_info_list := rt.LevelList
		for _, it := range route_info_list {
			// 这么做是因为产品会调整路线中的关卡
			if it.Status {
				info, err := data_access.TxGetRouteStatByLevelId(db_proxy, eid, rid, it.Did)
				if err != nil {
					log.Error("GetRouteStat error, err = %v, eid = %s, rid = %d", err, eid, rid)
					return err
				}
				if len(info.FEventId) == 0 {
					orid = orid + 1
					route_stat := data_access.RouteStat{
						FEventId:    eid,
						FRouteId:    rid,
						FOrderId:    orid,
						FLevelId:    it.Did,
						FFinishTime: nt_date,
						FStatus:     0,
						FCreateTime: nt_date,
						FModifyTime: nt_date,
					}
					if err = business_access.InsertRouteStat(db_proxy, route_stat); err != nil {
						log.Error("InsertRouteStat error, err = %v, route_stat = %v", err, route_stat)
						return err
					}
				}
			}
		}
	}

	max_order_id, _, err := data_access.TxGetRouteStatMaxInsertPoint(db_proxy, eid, rid)
	if err != nil {
		log.Error("TxGetRouteStatMaxInsertPoint error, err = %v, eid = %s, rid = %d", err, eid, rid)
		return err
	}

	if orid < max_order_id {
		if err := AddEventRouteStat(db_proxy, eid, nt_date, rid, orid); err != nil {
			log.Error("AddEventRouteStat error, err = %v", err)
			return err
		}
	}
	if err := business_access.UpdateRouteStatStatusPointFinishTime(db_proxy, eid, rid, orid, 1, nt_date); err != nil {
		log.Error("UpdateRouteStatStatusPointFinishTime error, err = %v", err)
		return err
	}

	// 转跳关卡的时候，锁定当前关卡的所有红包和公益包，后续要做成将红包全部奖励给完成关卡的人
	//row_size, err := business_access.UpdateRedPacketReceiveOidRefundByEidCP(db_proxy, eid, orid)
	//if err != nil {
	//	log.Error("UpdateRedPacketReceiveOidRefundByEidCP error, err = %v, eid = %s, cp = %d", err, eid, orid)
	//	return err
	//}
	//log.Info("affect row size = %d, eid = %s, cp = %d", row_size, eid, orid)
	row_size, err := business_access.UpdatePacketReceiveOidSenderByEidCP(db_proxy, eid, orid)
	if err != nil {
		log.Error("UpdatePacketReceiveOidSenderByEidCP error, err = %v, eid = %s, cp = %d", err, eid, orid)
		return err
	}
	log.Info("affect row size = %d, eid = %s, cp = %d", row_size, eid, orid)

	return nil
}
