//
package fundmgr

import (
	"context"
	"fmt"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	pb "git.code.oa.com/gongyi/donate_steps/api/fundmgr"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/connmgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/polarisclient"
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

func GetUserMatchDonate(oid string) (*pb.GetUserMatchDonateResponse, error) {
	// create request
	req := &pb.GetUserMatchDonateRequest{
		Oid: oid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetUserMatchDonateResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetUserMatchDonate",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
	svcReq.Method = "GetUserMatchDonate"
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

func GetUserWeekMatchRecord(oid, activityID, start, end string) (*pb.GetUserWeekMatchRecordResponse, error) {
	// create request
	req := &pb.GetUserWeekMatchRecordRequest{
		Oid:        oid,
		ActivityId: activityID,
		Start:      start,
		End:        end,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetUserWeekMatchRecordResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetUserWeekMatchRecord",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
			log.Error("%v - decode rawRsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
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
	svcReq.Method = "GetUserWeekMatchRecord"
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

func GetUserMatchRecordByOffset(oid, activityID string, offset, size int) (*pb.GetUserMatchRecordByOffsetResponse, error) {
	// create request
	req := &pb.GetUserMatchRecordByOffsetRequest{
		Oid:        oid,
		ActivityId: activityID,
		Offset:     int32(offset),
		Size:       int32(size),
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetUserMatchRecordByOffsetResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetUserMatchRecordByOffset",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
			log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
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
	svcReq.Method = "GetUserMatchRecordByOffset"
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

func UserMatch(oid, activityID string, steps int64) (*pb.UserMatchResponse, error) {
	// create request
	req := &pb.UserMatchRequest{
		Oid:        oid,
		ActivityId: activityID,
		Steps:      int32(steps),
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.UserMatchResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "UserMatch",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
			log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
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
	svcReq.Method = "UserMatch"
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

func GetUserTodayMatch(oid, activityID string) (*pb.GetUserTodayMatchResponse, error) {
	// create request
	req := &pb.GetUserTodayMatchRequest{
		Oid:        oid,
		ActivityId: activityID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetUserTodayMatchResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetUserTodayMatch",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
			log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}
	// get one instance from polaris
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
	svcReq.Method = "GetUserTodayMatch"
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

func GetActivityMatchRank(activityID string, offset, size int32) (*pb.GetActivityMatchRankResponse, error) {
	// create request
	req := &pb.GetActivityMatchRankRequest{
		ActivityId: activityID,
		Offset:     offset,
		Size:       size,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetActivityMatchRankResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetActivityMatchRank",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
			log.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
			return nil, err
		}
		log.Info("%v - transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return rsp, nil
	}
	// get one instance from polaris
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
	svcReq.Method = "GetActivityMatchRank"
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

func GetActivityMatchInfo(activityID string) (*pb.GetActivityMatchInfoResponse, error) {
	// create request
	req := &pb.GetActivityMatchInfoRequest{
		ActivityId: activityID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	methodName := "GetActivityMatchInfo"
	rsp := &pb.GetActivityMatchInfoResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: methodName,
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
	svcReq.Method = methodName
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

// GetCompanyMatchRank ...
func GetCompanyMatchRank(offset, size int) (*pb.GetCompanyMatchRankResponse, error) {
	// create request
	req := &pb.GetCompanyMatchRankRequest{
		Offset: int32(offset),
		Size:   int32(size),
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	method := "GetCompanyMatchRank"
	rsp := &pb.GetCompanyMatchRankResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: method,
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.fundmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
	svcReq.Method = method
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
