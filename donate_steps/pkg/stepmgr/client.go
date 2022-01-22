//
package stepmgr

import (
	"context"
	"fmt"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/api/metadata"
	pb "git.code.oa.com/gongyi/donate_steps/api/stepmgr"
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

func InitEdProxy(flag bool, pollInit, pollIdle, pollPeak int32, addr string) error {
	if flag {
		if err := gmclient.InitGRPCClient(pollInit, pollIdle, pollPeak); err != nil {
			log.Error("init edproxy err:%v", err)
			return err
		}
		gmclient.SetEdProxyAddr(addr)
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

func SetUserSteps(oid string, steps map[string]int64, background bool) (*pb.SetUsersStepsResponse, error) {
	if len(steps) == 0 {
		return &pb.SetUsersStepsResponse{
			Header: &metadata.CommonHeader{},
		}, nil
	}
	// create request
	var body = make(map[string]*metadata.UserSteps)
	var detail = make(map[string]int32)
	for key, value := range steps {
		detail[key] = int32(value)
	}
	body[oid] = &metadata.UserSteps{
		Oid:   oid,
		Steps: detail,
	}
	req := &pb.SetUsersStepsRequest{
		UserStep:   body,
		Background: background,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.SetUsersStepsResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "SetUsersSteps",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.stepmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
	svcReq.Method = "SetUsersSteps"
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

func GetUserSteps(oid string) (*pb.GetUsersStepsResponse, error) {
	// create request
	req := &pb.GetUsersStepsRequest{
		UserIds: []string{oid},
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetUsersStepsResponse{}
	// EdProxy mode
	if edProxyMode {
		edReq := &mygrpc.EdReq{
			TraceId:       uuid.New().String(),
			Namespace:     polarisConf.PolarisEnv,
			ServiceName:   polarisConf.PolarisServiceName,
			ServiceMethod: "GetUsersSteps",
			Body:          data,
		}
		log.Info("%v - transID: %v, request: %s", util.GetCallee(), edReq.TraceId, edReq)
		// call rpc method
		edRes, err := gmclient.CallEdProxy("yqz.stepmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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
	svcReq.Method = "GetUsersSteps"
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

// GetUserPkProfile ...
func GetUsersPkProfile(oid []string) (*pb.GetUsersPkProfileResponse, error) {
	// create request
	method := "GetUsersPkProfile"
	req := &pb.GetUsersPkProfileRequest{
		UserIds: oid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetUsersPkProfileResponse{}
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
		edRes, err := gmclient.CallEdProxy("yqz.stepmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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

// GetPkInteract ...
func GetPkInteract(oid, ToOid string) (*pb.GetPkInteractResponse, error) {
	// create request
	method := "GetPkInteract"
	req := &pb.GetPkInteractRequest{
		UserId:   oid,
		ToUserId: ToOid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetPkInteractResponse{}
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
		edRes, err := gmclient.CallEdProxy("yqz.stepmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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

func SetPkInteract(oid, ToOid string, interact pb.SetPkInteractRequest_InteractType, op pb.SetPkInteractRequest_InteractOps) (*pb.SetPkInteractResponse, error) {
	// create request
	method := "SetPkInteract"
	req := &pb.SetPkInteractRequest{
		UserId:   oid,
		ToUserId: ToOid,
		Interact: interact,
		Ops:      op,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.SetPkInteractResponse{}
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
		edRes, err := gmclient.CallEdProxy("yqz.stepmgr", edReq, time.Duration(time.Millisecond*TIMEOUT_MS))
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

func SetActivePk(oid, toOid string) (*pb.SetActivePkResponse, error) {
	// create request
	method := "SetActivePk"
	req := &pb.SetActivePkRequest{
		UserId:   oid,
		ToUserId: toOid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.SetActivePkResponse{}
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

func SetResponsePk(oid, toOid string) (*pb.SetResponsePkResponse, error) {
	// create request
	method := "SetResponsePk"
	req := &pb.SetResponsePkRequest{
		UserId:   oid,
		ToUserId: toOid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.SetResponsePkResponse{}
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

func GetPassivePk(oid string) (*pb.GetPassivePkResponse, error) {
	// create request
	method := "GetPassivePk"
	req := &pb.GetPassivePkRequest{
		UserId: oid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetPassivePkResponse{}
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

func GetPkNotification(oid string) (*pb.GetPkNotificationResponse, error) {
	// create request
	method := "GetPkNotification"
	req := &pb.GetPkNotificationRequest{
		UserId: oid,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Error("%v - proto.Marshal err: %v", util.GetCallee, err.Error())
		return nil, err
	}
	rsp := &pb.GetPkNotificationResponse{}
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
