package activitymgr

import (
	"context"
	"fmt"
	"time"

	"git.code.oa.com/gongyi/donate_steps/pkg/polarisclient"

	"git.code.oa.com/gongyi/agw/log"
	pb "git.code.oa.com/gongyi/donate_steps/api/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/api/metadata"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/connmgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	gmclient "git.code.oa.com/gongyi/gomore/clients/grpc_client"
	mygrpc "git.code.oa.com/gongyi/gomore/grpc/proto"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const TIMEOUT_MS = 3000

var polarisClient *polarisclient.PolarisClient
var polarisConf *polarisclient.PolarisInfo
var edProxyMode = false

func InitEdProxy(conf common.EdProxyConfig) error {
	if conf.Flag {
		if err := gmclient.InitGRPCClient(int32(conf.PollInit), int32(conf.PollIdle), int32(conf.PollPeak)); err != nil {
			log.Error("init edproxy err:%v", err)
			return err
		}
		gmclient.SetEdProxyAddr(conf.Addr)
		edProxyMode = true
	}
	return nil
}

func InitPolarisClient(conf *polarisclient.PolarisInfo) error {
	var err error
	polarisClient, err = polarisclient.NewClient()
	if err != nil {
		log.Error("init polaris client err:%v", err)
		return err
	}
	polarisConf = conf
	return nil
}

func ClosePolarisClient() {
	polarisClient.Close()
}

func serialCreateActivityRequest(request *common.CreateActivityPlatformRequest) *pb.CreateActivityRequest {
	activity := &metadata.Activity{
		ActivityCreator: request.Activity.Creator,
		Type:            metadata.Activity_CreateType(request.Activity.Type),
		Desc:            request.Activity.Name,
		Slogan:          request.Activity.Slogan,
		RouteId:         request.Activity.RouteId,
		Bgpic:           request.Activity.Bgpic,
		StartTime:       request.Activity.StartTime,
		EndTime:         request.Activity.EndTime,
		TeamMode:        int32(request.Activity.TeamMode),
		TeamOff:         int32(request.Activity.TeamOff),
		TeamMemberLimit: int32(request.Activity.TeamMemberLimit),
		DefaultTeams:    request.Activity.DefaultTeams,
		OrgName:         request.Activity.OrgName,
		OrgHead:         request.Activity.OrgHead,
		ShowSponsor:     int32(request.Activity.ShowSponsor),
		MatchOff:        int32(request.Activity.MatchOff),
		Rule:            request.Activity.Rule,
		Color:           request.Activity.Color,
		CompanyId:       request.Activity.CompanyID,
		SuppCoupons:     request.Activity.SuppCoupons,
		WhiteType:       metadata.Activity_WhiteType(request.Activity.WhiteType),
		CreateTime:      request.Activity.CreateTime,
		Cover:           request.Activity.Cover,
		RelationDesc:    request.Activity.RelationDesc,
		ForwardPic:      request.Activity.ForwardPic,
	}

	// 配捐开始和结束时间, 默认采用活动开始和结束时间,
	// 支持自定义配捐时间, 配捐时间必须是活动时间的子集
	matchStartTime := request.Activity.StartTime
	matchEndTime := request.Activity.EndTime
	if len(request.Match.StartTime) > 0 {
		matchStartTime = request.Match.StartTime
	}
	if len(request.Match.EndTime) > 0 {
		matchEndTime = request.Match.EndTime
	}

	matchInfo := &metadata.MatchInfo{
		FCompanyId:   request.Match.CompanyId,
		FPid:         request.Match.Pid,
		FTargetFund:  request.Match.TargetFund,
		FStartTime:   matchStartTime,
		FEndTime:     matchEndTime,
		FMatchMode:   int32(request.Match.MatchMode),
		FMatchModeV2: metadata.MatchInfo_MatchMode(request.Match.MatchMode),
		FRemindFund:  request.Match.RemindFund,
		FCertificate: request.Match.Certificate,
		FLoveCoupon:  request.Activity.SuppCoupons, //支持爱心券，透传至配捐结构体里
	}

	buffMeta := &metadata.BuffMeta{
		BasicFund: int32(request.Match.BasicFund),
		BasicWave: int32(request.Match.BasicWave),
		BuffFund:  int32(request.Match.BuffFund),
		BuffWave:  int32(request.Match.BuffWave),
	}
	buffComboRule := &metadata.BuffComboRule{
		BuffMeta:      buffMeta,
		BuffThreshold: int32(request.Match.BuffThreshold),
	}
	buffPercentRule := &metadata.BuffPercentRule{
		BuffMeta:    buffMeta,
		BuffPercent: int32(request.Match.BuffPercent),
	}
	stepRelativeRule := &metadata.StepRelativeRule{
		MinSteps: int32(request.Match.MinSteps),
		MaxSteps: int32(request.Match.MaxSteps),
		MinMatch: int32(request.Match.MinMatch),
		MaxMatch: int32(request.Match.MaxMatch),
	}
	matchRule := &metadata.MatchRule{
		MatchQuota:       int32(request.Match.StepQuota),
		RuleType:         metadata.MatchRule_RuleType(request.Match.RuleType),
		BuffComboRule:    buffComboRule,
		BuffPercentRule:  buffPercentRule,
		RemainMatchRules: &metadata.RemainMatchRules{RemainMatchRules: make([]*metadata.RemainMatchRule, 0)},
		StepRelativeRule: stepRelativeRule,
	}

	return &pb.CreateActivityRequest{
		Oid:       request.Oid,
		Appid:     request.Appid,
		UnionId:   request.UniId,
		Activity:  activity,
		MatchInfo: matchInfo,
		MatchRule: matchRule,
		WhiteList: request.Activity.WhiteList,
	}
}

func CreateActivity(request *common.CreateActivityPlatformRequest) (*pb.CreateActivityResponse, error) {
	// create request
	req := serialCreateActivityRequest(request)
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.CreateActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "CreateActivity",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "CreateActivity"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}
	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func QueryActivity(aid string) (*pb.QueryActivityResponse, error) {
	// create request
	req := &pb.QueryActivityRequest{
		Aid: aid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.QueryActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "QueryActivity",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "QueryActivity"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func UpdateActivityStatus(aid string, status metadata.Activity_Status) (*pb.UpdateActivityStatusResponse, error) {
	// create request
	req := &pb.UpdateActivityStatusRequest{
		Aid:    aid,
		Status: int32(status),
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.UpdateActivityStatusResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "UpdateActivityStatus",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "UpdateActivityStatus"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func JoinActivity(oid, aid, phoneNum, appid, unionId string) (*pb.JoinActivityResponse, error) {
	req := &pb.JoinActivityRequest{
		Oid:      oid,
		Aid:      aid,
		Appid:    appid,
		UniId:    unionId,
		PhoneNum: phoneNum,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.JoinActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "JoinActivity",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*3000)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "JoinActivity"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}
	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func JoinTeam(oid, aid, tid, phoneNum string, join bool, appid, unionId string) (*pb.JoinTeamResponse, error) {
	req := &pb.JoinTeamRequest{
		Oid:      oid,
		Aid:      aid,
		Tid:      tid,
		Join:     join,
		Appid:    appid,
		UniId:    unionId,
		PhoneNum: phoneNum,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.JoinTeamResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "JoinTeam",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "JoinTeam"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}
	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func ActivityTeamRank(oid, aid string, page, size int64, rank_type string) (*pb.ActivityTeamRankResponse, error) {
	req := &pb.ActivityTeamRankRequest{
		Aid:      aid,
		Page:     page,
		Size:     size,
		RankType: rank_type,
		Oid:      oid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.ActivityTeamRankResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "ActivityTeamRank",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "ActivityTeamRank"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}
	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func ActivityUserRank(oid, aid string, page, size int64, rank_type string) (*pb.ActivityUserRankResponse, error) {
	req := &pb.ActivityUserRankRequest{
		Aid:      aid,
		Page:     page,
		Size:     size,
		RankType: rank_type,
		Oid:      oid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.ActivityUserRankResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "ActivityUserRank",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*3000)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "ActivityUserRank"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}
	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func CreateTeam(request *common.CreateTeamRequest) (*pb.CreateTeamResponse, error) {
	// create request
	req := &pb.CreateTeamRequest{
		Team: &metadata.Team{
			ActivityId:  request.Aid,
			TeamType:    int32(request.Team.TeamType),
			TeamDesc:    request.Team.TeamName,
			TeamCreator: request.Team.TeamCreator,
			TeamFlag:    metadata.Team_TeamFlag(request.Team.TeamFlag),
		},
		Oid:     request.Oid,
		Appid:   request.Appid,
		UnionId: request.UniId,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.CreateTeamResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "CreateTeam",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "CreateTeam"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}
	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func QueryTeams(request *common.GetTeamsRequest) (*pb.QueryTeamsResponse, error) {
	// create request
	req := &pb.QueryTeamsRequest{
		Aid:     request.Aid,
		Oid:     request.Oid,
		TeamIds: request.TeamIds,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.QueryTeamsResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "QueryTeams",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "QueryTeams"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func TeamUserRank(aid, tid, oid string, page, size int64) (*pb.TeamUserRankResponse, error) {
	req := &pb.TeamUserRankRequest{
		Aid:  aid,
		Oid:  oid,
		Tid:  tid,
		Page: page,
		Size: size,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.TeamUserRankResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "TeamUserRank",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "TeamUserRank"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}
	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func QueryActivityUserStats(aid, oid string) (*pb.QueryActivityUserStatsResponse, error) {
	// create request
	req := &pb.QueryActivityUserStatsRequest{
		Aid: aid,
		Oid: oid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.QueryActivityUserStatsResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "QueryActivityUserStats",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "QueryActivityUserStats"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

// 查询用户参与的活动列表
func QueryUserJoinActivity(oid string, page, size int64) (*pb.QueryUserJoinActivityResponse, error) {
	// create request
	req := &pb.QueryUserJoinActivityRequest{
		Oid:  oid,
		Page: page,
		Size: size,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.QueryUserJoinActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "QueryUserJoinActivity",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "QueryUserJoinActivity"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

// QueryUserJoinActivityNew ...
func QueryUserJoinActivityNew(oid string, page, size int64) (*pb.QueryUserJoinActivityResponse, error) {
	// create request
	req := &pb.QueryUserJoinActivityRequest{
		Oid:  oid,
		Page: page,
		Size: size,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.QueryUserJoinActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "QueryUserJoinActivityNew",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "QueryUserJoinActivityNew"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

// 查询小队统计
func QueryTeamStats(aid, teamId string) (*pb.QueryTeamStatsResponse, error) {
	// create request
	req := &pb.QueryTeamStatsRequest{
		Aid:    aid,
		TeamId: teamId,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.QueryTeamStatsResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "QueryTeamStats",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "QueryTeamStats"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

// 查询用户参与的活动列表
func QueryUserCreateActivity(oid string, page, size int64) (*pb.QueryUserJoinActivityResponse, error) {
	// create request
	req := &pb.QueryUserJoinActivityRequest{
		Oid:  oid,
		Page: page,
		Size: size,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.QueryUserJoinActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "QueryUserCreateActivity",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "QueryUserCreateActivity"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func GetActivityList(offset, size int32, activityID string, status int32, name string) (*pb.GetActivityListResponse, error) {
	// create request
	req := &pb.GetActivityListRequest{
		Offset:     offset,
		Size:       size,
		ActivityId: activityID,
		Status:     status,
		Name:       name,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetActivityListResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetActivityList",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "GetActivityList"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func GetActivityTeamList(offset, size int32, activityID string, tType int32) (*pb.GetActivityTeamListResponse, error) {
	// create request
	req := &pb.GetActivityTeamListRequest{
		ActivityId: activityID,
		Offset:     offset,
		Size:       size,
		Type:       tType,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetActivityTeamListResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetActivityTeamList",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "GetActivityTeamList"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func UpdateActivity(activityID string, activity *metadata.Activity, matchInfo *metadata.MatchInfo, matchRule *metadata.MatchRule, whiteList []string) (*pb.UpdateActivityResponse, error) {
	// create request
	req := &pb.UpdateActivityRequest{
		ActivityId: activityID,
		Activity:   activity,
		MatchInfo:  matchInfo,
		MatchRule:  matchRule,
		WhiteList:  whiteList,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.UpdateActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "UpdateActivity",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "UpdateActivity"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func UpdateActivityState(activityID string, state int32) (*pb.UpdateActivityStateResponse, error) {
	// create request
	req := &pb.UpdateActivityStateRequest{
		ActivityId: activityID,
		State:      state,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.UpdateActivityStateResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "UpdateActivityState",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "UpdateActivityState"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func UpdateActivityTeam(activityID, teamID, teamName string) (*pb.UpdateActivityTeamResponse, error) {
	// create request
	req := &pb.UpdateActivityTeamRequest{
		ActivityId: activityID,
		TeamId:     teamID,
		TeamName:   teamName,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.UpdateActivityTeamResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "UpdateActivityTeam",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "UpdateActivityTeam"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func DeleteActivity(activityID string, opType int32, operator string) (*pb.DeleteActivityResponse, error) {
	// create request
	req := &pb.DeleteActivityRequest{
		ActivityId: activityID,
		Operator:   operator,
		OpType:     pb.DeleteActivityRequest_Ops(opType),
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.DeleteActivityResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "DeleteActivity",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "DeleteActivity"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func DeleteActivityTeam(activityID, teamID string) (*pb.DeleteActivityTeamResponse, error) {
	// create request
	req := &pb.DeleteActivityTeamRequest{
		ActivityId: activityID,
		TeamId:     teamID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.DeleteActivityTeamResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "DeleteActivityTeam",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "DeleteActivityTeam"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func ModifyTeam(oid, activityID, teamID, teamDesc string, teamType int32) (*pb.ModifyTeamResponse, error) {
	// create request
	req := &pb.ModifyTeamRequest{
		Oid:        oid,
		ActivityId: activityID,
		TeamId:     teamID,
		TeamDesc:   teamDesc,
		TeamType:   teamType,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.ModifyTeamResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "ModifyTeam",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "ModifyTeam"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func GetTeamsByUsers(aid string, oids []string) (*pb.GetTeamsByUsersResponse, error) {
	// create request
	req := &pb.GetTeamsByUsersRequest{
		Aid:  aid,
		Oids: oids,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetTeamsByUsersResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetTeamsByUsers",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "GetTeamsByUsers"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}

func ChangeTeamLeader(oid, activityID, teamID, operatorOid string) (*pb.ChangeTeamLeaderResponse, error) {
	// create request
	req := &pb.ChangeTeamLeaderRequest{
		Oid:         oid,
		ActivityId:  activityID,
		TeamId:      teamID,
		OperatorOid: operatorOid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.ChangeTeamLeaderResponse{}

	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "ChangeTeamLeader",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.activitymgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			log.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			log.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			log.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}

	// direct mode
	ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
	if nil != err {
		return nil, err
	}
	//connStr := fmt.Sprintf("%s:%d", common.Yqzconfig.FundMgr.Addr, common.Yqzconfig.FundMgr.Port)
	connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
	conn, err := connmgr.GetGrpcConn(connStr)
	if err != nil {
		log.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
		return nil, err
	}
	client := mygrpc.NewMserviceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
	defer cancel()

	// create request
	var svcReq mygrpc.SvcReq
	svcReq.Method = "ChangeTeamLeader"
	svcReq.TraceId = uuid.New().String()
	svcReq.Body = data

	// 用户rpc方法
	log.Info("%v - transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
	svcRes, err := client.Call(ctx, &svcReq)
	if err != nil {
		log.Error("%v - grpc call err: %v", util.GetCallee(), err.Error())
		// close conn and establish again
		connmgr.CloseGrpcConn(connStr)
		return nil, err
	}

	err = proto.Unmarshal(svcRes.Body, rsp)
	if err != nil {
		log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
		return nil, err
	}

	log.Info("%v - transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	return rsp, nil
}
