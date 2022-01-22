//
//

package common

import (
	"encoding/json"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	XGYProto "git.code.oa.com/gongyi/gongyi_base/proto/gy_agent"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

var KafkaRedisLog *gyredis.MyLog
var KafkaRedisConf *gyredis.RedisConf
var KafkaSender *gyredis.RedisClient

func KafkaSenderInit() {
	KafkaRedisLog, _ = gyredis.NewMyLog("DEBUG", "../log/kafka_sender.log", 4, 10240000)
	KafkaRedisConf = gyredis.NewDefaultRedisConf()
	KafkaRedisConf.Addr = "127.0.0.1:6379"
	var err error
	KafkaSender, err = gyredis.NewRedisClient(KafkaRedisConf, KafkaRedisLog)
	if err != nil {
		KafkaRedisLog.Error("KafkaSenderInit failed",
			zap.String("config = ", KafkaRedisConf.Addr),
			zap.String("init error = ", err.Error()))
		panic("init redis error")
	}
}

func KafkaSendMsg(msg []byte) error {
	_, err := KafkaSender.LPush("gy_feeds_0", msg).Result()
	if err != nil {
		log.Error("lpush error, err = %v, msg = %s", err, string(msg))
		return err
	}
	return nil
}

func AddKeySValField(fields []*XGYProto.KeyVal, key string, val []byte) []*XGYProto.KeyVal {
	var field_key_val XGYProto.KeyVal
	field_key_val.Key = &key
	field_key_val.Sval = val
	fields = append(fields, &field_key_val)
	return fields
}

func AddKeyIValField(fields []*XGYProto.KeyVal, key string, val float64) []*XGYProto.KeyVal {
	var field_key_val XGYProto.KeyVal
	field_key_val.Key = &key
	field_key_val.Ival = &val
	fields = append(fields, &field_key_val)
	return fields
}

// async cover step
type AsyncCoverStep struct {
	Oid      string
	DateStep map[string]int64
}

func GenerateStep(oid, unid string, steps map[string]int64) ([]byte, error) {
	var msg XGYProto.ReportMsg
	msg.MsgType = XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM.Enum()
	metrics := "metrics"
	msg.Metrics = &metrics
	topic := "yqz-update-step"
	msg.Topic = &topic
	msg.Key = &oid
	nt_date := time.Now().Format("2006-01-02 15:04:05")
	msg.Dt = &nt_date
	msg.Fields = make([]*XGYProto.KeyVal, 0)
	msg.Fields = AddKeySValField(msg.Fields, "busi_type", []byte("YQZ_UpdateStep"))
	msg.Fields = AddKeySValField(msg.Fields, "oid", []byte(oid))
	msg.Fields = AddKeySValField(msg.Fields, "unid", []byte(unid))
	date_step_json, err := json.Marshal(steps)
	if err != nil {
		log.Error("json marshal error, err = %v, date_step = %+v", err, steps)
		return nil, err
	}
	msg.Fields = AddKeySValField(msg.Fields, "date_step", date_step_json)
	log.Debug("msg = %+v", msg)
	return proto.Marshal(&msg)
}

/*
func GenerateCoverStep(input *AsyncCoverStep) ([]byte, error) {
	var msg XGYProto.ReportMsg
	msg.MsgType = XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM.Enum()
	metrics := "metrics"
	msg.Metrics = &metrics
	topic := "yqz-cover-step"
	msg.Topic = &topic
	msg.Key = &input.Oid
	nt_date := time.Now().Format("2006-01-02 15:04:05")
	msg.Dt = &nt_date
	msg.Fields = make([]*XGYProto.KeyVal, 0)
	msg.Fields = AddKeySValField(msg.Fields, "busi_type", []byte("YQZ_CoverStep"))
	msg.Fields = AddKeySValField(msg.Fields, "oid", []byte(input.Oid))
	date_step_json, err := json.Marshal(input.DateStep)
	if err != nil {
		log.Error("json marshal error, err = %v, date_step = %+v", err, input.DateStep)
		return nil, err
	}
	msg.Fields = AddKeySValField(msg.Fields, "date_step", date_step_json)
	log.Debug("msg = %+v", msg)
	return proto.Marshal(&msg)
}
*/

type WeekChallenge struct {
	Oid  string
	Step int64
	Rank int64
	Num  int64
	Time string
}

func GenerateWeekChallenge(input *WeekChallenge) ([]byte, error) {
	var msg XGYProto.ReportMsg
	msg.MsgType = XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM.Enum()
	metrics := "metrics"
	msg.Metrics = &metrics
	topic := "yqz-week-challenge"
	msg.Topic = &topic
	msg.Key = &input.Oid
	nt_date := time.Now().Format("2006-01-02 15:04:05")
	msg.Dt = &nt_date
	msg.Fields = make([]*XGYProto.KeyVal, 0)
	msg.Fields = AddKeySValField(msg.Fields, "busi_type", []byte("YQZ_WeekChallenge"))
	msg.Fields = AddKeySValField(msg.Fields, "oid", []byte(input.Oid))
	msg.Fields = AddKeyIValField(msg.Fields, "step", float64(input.Step))
	msg.Fields = AddKeyIValField(msg.Fields, "rank", float64(input.Rank))
	msg.Fields = AddKeyIValField(msg.Fields, "num", float64(input.Num))
	msg.Fields = AddKeySValField(msg.Fields, "time", []byte(input.Time))
	log.Debug("GenerateWeekChallenge msg = %+v", msg)
	return proto.Marshal(&msg)
}

type RedEnvelopeDonation struct {
	Trid    string
	Eid     string
	Oid     string
	Pid     int
	Fund    int64
	Receive int64
	Time    string
}

func GenerateRedEnvelopeDonationData(input *RedEnvelopeDonation) ([]byte, error) {
	var msg XGYProto.ReportMsg
	msg.MsgType = XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM.Enum()
	metrics := "metrics"
	msg.Metrics = &metrics
	topic := "yqz-red-envelope-donation"
	msg.Topic = &topic
	msg.Key = &input.Oid
	nt_date := time.Now().Format("2006-01-02 15:04:05")
	msg.Dt = &nt_date
	msg.Fields = make([]*XGYProto.KeyVal, 0)
	msg.Fields = AddKeySValField(msg.Fields, "busi_type", []byte("YQZ_RedEnvelopeDonation"))
	msg.Fields = AddKeySValField(msg.Fields, "trid", []byte(input.Trid))
	msg.Fields = AddKeySValField(msg.Fields, "eid", []byte(input.Eid))
	msg.Fields = AddKeySValField(msg.Fields, "oid", []byte(input.Oid))
	msg.Fields = AddKeyIValField(msg.Fields, "pid", float64(input.Pid))
	msg.Fields = AddKeyIValField(msg.Fields, "fund", float64(input.Fund))
	msg.Fields = AddKeyIValField(msg.Fields, "receive", float64(input.Receive))
	msg.Fields = AddKeySValField(msg.Fields, "time", []byte(input.Time))
	log.Debug("GenerateRedEnvelopeDonationData msg = %+v", msg)
	return proto.Marshal(&msg)
}

type PoiArriveInfo struct {
	Oid       string
	Week      string
	PoiName   string
	RouteName string
	ArriveNum int
}

func GeneratePoiArriveData(input *PoiArriveInfo) ([]byte, error) {
	var msg XGYProto.ReportMsg
	msg.MsgType = XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM.Enum()
	metrics := "metrics"
	msg.Metrics = &metrics
	topic := "yqz-general-msg-push"
	msg.Topic = &topic
	msg.Key = &input.Oid
	nt_date := time.Now().Format("2006-01-02 15:04:05")
	msg.Dt = &nt_date
	msg.Fields = make([]*XGYProto.KeyVal, 0)
	msg.Fields = AddKeySValField(msg.Fields, "busi_type", []byte("YQZ_PoiArrive"))
	msg.Fields = AddKeySValField(msg.Fields, "oid", []byte(input.Oid))
	msg.Fields = AddKeySValField(msg.Fields, "week", []byte(input.Week))
	msg.Fields = AddKeySValField(msg.Fields, "poi_name", []byte(input.PoiName))
	msg.Fields = AddKeySValField(msg.Fields, "route_name", []byte(input.RouteName))
	msg.Fields = AddKeyIValField(msg.Fields, "arrive_num", float64(input.ArriveNum))
	log.Debug("GeneratePoiArriveData msg = %+v", msg)
	return proto.Marshal(&msg)
}
