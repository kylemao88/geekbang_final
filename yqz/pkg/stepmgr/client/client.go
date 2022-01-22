// //
package client

import (
	"context"
	"fmt"
	"time"

	gmclient "git.code.oa.com/gongyi/gomore/clients/grpc_client"
	mygrpc "git.code.oa.com/gongyi/gomore/grpc/proto"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	pb "git.code.oa.com/gongyi/yqz/api/stepmgr"
	"git.code.oa.com/gongyi/yqz/pkg/common/connmgr"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/polarisclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/proxyclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const TIMEOUT_MS = 3000

var polarisClient *polarisclient.PolarisClient
var polarisConf *polarisclient.PolarisInfo
var clientType proxyclient.ClientType

func InitClient(proxy *proxyclient.ProxyConfig, service *proxyclient.ServiceConfig) error {
	polarisConf = &polarisclient.PolarisInfo{
		PolarisEnv: service.PolarisEnv,
	}
	if proxy.Flag {
		switch proxy.ProxyType {
		case "Sidecar":
			if err := proxyclient.InitSidecarPorxy(proxy); err != nil {
				logger.Error("init sidecar proxy err:%v", err)
				return err
			}
			clientType = proxyclient.Type_Sidecar
		case "Edproxy":
			if err := proxyclient.InitEdProxy(proxy); err != nil {
				logger.Error("init edproxy err:%v", err)
				return err
			}
			clientType = proxyclient.Type_Edproxy
		default:
			logger.Error("unknow type: %v", proxy.ProxyType)
			return fmt.Errorf("unknow type: %v", proxy.ProxyType)
		}
		polarisConf.PolarisServiceName = service.PolarisAddr
	} else {
		var err error
		polarisClient, err = polarisclient.NewClient()
		if err != nil {
			logger.Error("init polaris client err:%v", err)
			return err
		}
		polarisConf.PolarisServiceName = service.FullPolarisAddr
		clientType = proxyclient.Type_Polaris
	}
	return nil
}

/*
// InitPolarisClient init polaris
func InitPolarisClient(conf *polarisclient.PolarisInfo) error {
	var err error
	polarisClient, err = polarisclient.NewClient()
	if err != nil {
		logger.Error("init polaris client err:%v", err)
		return err
	}
	polarisConf = conf
	clientType = proxyclient.Type_Polaris
	return nil
}
*/

// CloseClient ...
func CloseClient() {
	polarisClient.Close()
}

// SetUsersSteps set user steps into db and cache
func SetUsersSteps(steps map[string]*metadata.UserSteps, background bool) error {
	// create request
	req := &pb.SetUsersStepsRequest{
		UserStep:   steps,
		Background: background,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		logger.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return err
	}
	rsp := &pb.SetUsersStepsResponse{}
	traceID := uuid.New().String()
	switch clientType {
	case proxyclient.Type_Edproxy:
		edReq := &mygrpc.EdReq{
			TraceId:       traceID,
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "SetUsersSteps",
			Body:          data,
		}
		logger.Debug("%v - [CallEdProxy] transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call edproxy method
		edRes, err := gmclient.CallEdProxy("yqz.stepmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			logger.Error("[CallEdProxy] call failed: %v", err)
			return err
		}
		if edRes.Errcode != 0 {
			logger.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			logger.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return err
		}
		logger.Debug("%v - [CallEdProxy] transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
		return nil
	case proxyclient.Type_Polaris:
		// direct mode
		ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
		if nil != err {
			return err
		}
		connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
		conn, err := connmgr.GetGrpcConn(connStr)
		if err != nil {
			logger.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
			return err
		}
		client := mygrpc.NewMserviceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
		defer cancel()
		// create request
		svcReq := &mygrpc.SvcReq{
			TraceId: traceID,
			Method:  "SetUsersSteps",
			Body:    data,
		}
		logger.Debug("%v - [Direct] transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
		svcRes, err := client.Call(ctx, svcReq)
		if err != nil {
			logger.Error("%v - [Direct] grpc call err: %v", util.GetCallee(), err.Error())
			// close conn and establish again
			connmgr.CloseGrpcConn(connStr)
			return err
		}
		if svcRes.Errcode != 0 {
			logger.Error("[Direct] call failed: errCode[%d], body[%v]", svcRes.Errcode, string(svcRes.Body))
			return fmt.Errorf("[Direct] call failed: errCode[%d], body[%v]", svcRes.Errcode, svcRes.Body)
		}
		err = proto.Unmarshal(svcRes.Body, rsp)
		if err != nil {
			logger.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
			return err
		}
		logger.Debug("%v - [Direct] transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	case proxyclient.Type_Sidecar:
		svcReq := &mygrpc.SvcReq{
			TraceId:   traceID,
			Namespace: polarisConf.PolarisEnv,
			Name:      polarisConf.PolarisServiceName,
			Method:    "SetUsersSteps",
			Body:      data,
		}
		svcRes, err := gmclient.CallSidecar("yqz.stepmgr", svcReq, time.Duration(TIMEOUT_MS))
		if err != nil {
			logger.Error("[CallSidecar] call failed: %v", err)
			return err
		}
		if svcRes.Errcode != 0 {
			logger.Error("[CallSidecar] call failed: errCode[%d], body[%v]", svcRes.Errcode, string(svcRes.Body))
			return fmt.Errorf("[CallSidecar] call failed: errCode[%d], body[%v]", svcRes.Errcode, svcRes.Body)
		}
		logger.Debug("%v - [CallSidecar] transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, svcReq)
		// handle response
		err = proto.Unmarshal(svcRes.Body, rsp)
		if err != nil {
			logger.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(svcRes.Body), err.Error())
			return err
		}
		logger.Debug("%v - [CallSidecar] transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	default:
		logger.Error("unknow client type: %v", clientType.String())
		return fmt.Errorf("unknow client type: %v", clientType.String())
	}
	return nil
}

// GetUsersSteps get users recently 30 days steps
func GetUsersSteps(userIDs []string) (*pb.GetUsersStepsResponse, error) {
	// create request
	req := &pb.GetUsersStepsRequest{
		UserIds: userIDs,
	}
	return getUsersStepsCore(req)
}

// GetUsersSteps get users range steps from db
func GetUsersRangeSteps(userIDs []string, start, end string) (*pb.GetUsersStepsResponse, error) {
	// create request
	req := &pb.GetUsersStepsRequest{
		UserIds:   userIDs,
		RangeFlag: true,
		StartTime: start,
		EndTime:   end,
	}
	return getUsersStepsCore(req)
}

func getUsersStepsCore(req *pb.GetUsersStepsRequest) (*pb.GetUsersStepsResponse, error) {
	data, err := proto.Marshal(req)
	if err != nil {
		logger.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetUsersStepsResponse{}
	traceID := uuid.New().String()
	switch clientType {
	case proxyclient.Type_Edproxy:
		edReq := &mygrpc.EdReq{
			TraceId:       traceID,
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetUsersSteps",
			Body:          data,
		}
		logger.Debug("%v - [CallEdProxy] transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call edproxy method
		edRes, err := gmclient.CallEdProxy("yqz.stepmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
		if err != nil {
			logger.Error("[CallEdProxy] call failed: %v", err)
			return nil, err
		}
		if edRes.Errcode != 0 {
			logger.Error("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, string(edRes.Body))
			return nil, fmt.Errorf("[CallEdProxy] call failed: errCode[%d], body[%v]", edRes.Errcode, edRes.Body)
		}
		// handle response
		err = proto.Unmarshal(edRes.Body, rsp)
		if err != nil {
			logger.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(edRes.Body), err.Error())
			return nil, err
		}
		logger.Debug("%v - [CallEdProxy] transID: %v, response: %s", util.GetCallee(), edReq.TraceId, rsp)
	case proxyclient.Type_Polaris:
		// direct mode
		ins, err := polarisClient.GetOneInstance(polarisConf.PolarisServiceName, polarisConf.PolarisEnv)
		if nil != err {
			return nil, err
		}
		connStr := fmt.Sprintf("%s:%d", ins.GetHost(), ins.GetPort())
		conn, err := connmgr.GetGrpcConn(connStr)
		if err != nil {
			logger.Error("%v - connmgr.GetGrpcConn get conn err: %v", util.GetCallee(), err)
			return nil, err
		}
		client := mygrpc.NewMserviceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*TIMEOUT_MS)
		defer cancel()
		// create request
		svcReq := &mygrpc.SvcReq{
			TraceId: traceID,
			Method:  "GetUsersSteps",
			Body:    data,
		}
		logger.Debug("%v - [Direct] transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, req)
		svcRes, err := client.Call(ctx, svcReq)
		if err != nil {
			logger.Error("%v - [Direct] grpc call err: %v", util.GetCallee(), err.Error())
			// close conn and establish again
			connmgr.CloseGrpcConn(connStr)
			return nil, err
		}
		if svcRes.Errcode != 0 {
			logger.Error("[Direct] call failed: errCode[%d], body[%v]", svcRes.Errcode, string(svcRes.Body))
			return nil, fmt.Errorf("[Direct] call failed: errCode[%d], body[%v]", svcRes.Errcode, svcRes.Body)
		}
		err = proto.Unmarshal(svcRes.Body, rsp)
		if err != nil {
			logger.Error("%v - decode rsp err: %v", util.GetCallee(), err.Error())
			return nil, err
		}
		logger.Debug("%v - [Direct] transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	case proxyclient.Type_Sidecar:
		svcReq := &mygrpc.SvcReq{
			TraceId:   traceID,
			Namespace: polarisConf.PolarisEnv,
			Name:      polarisConf.PolarisServiceName,
			Method:    "GetUsersSteps",
			Body:      data,
		}
		svcRes, err := gmclient.CallSidecar("yqz.stepmgr", svcReq, time.Duration(TIMEOUT_MS))
		if err != nil {
			logger.Error("[CallSidecar] call failed: %v", err)
			return nil, err
		}
		if svcRes.Errcode != 0 {
			logger.Error("[CallSidecar] call failed: errCode[%d], body[%v]", svcRes.Errcode, string(svcRes.Body))
			return nil, fmt.Errorf("[CallSidecar] call failed: errCode[%d], body[%v]", svcRes.Errcode, svcRes.Body)
		}
		logger.Debug("%v - [CallSidecar] transID: %v, request: %s", util.GetCallee(), svcReq.TraceId, svcReq)
		// handle response
		err = proto.Unmarshal(svcRes.Body, rsp)
		if err != nil {
			logger.Error("%v - decode rsp: %v err: %v", util.GetCallee(), string(svcRes.Body), err.Error())
			return nil, err
		}
		logger.Debug("%v - [CallSidecar] transID: %v, response: %s", util.GetCallee(), svcReq.TraceId, rsp)
	default:
		logger.Error("unknow client type: %v", clientType.String())
		return nil, fmt.Errorf("unknow client type: %v", clientType.String())
	}
	return rsp, nil
}
