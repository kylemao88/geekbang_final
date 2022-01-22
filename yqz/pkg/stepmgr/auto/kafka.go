//
package auto

import (
	"encoding/json"

	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	XGYProto "git.code.oa.com/gongyi/gongyi_base/proto/gy_agent"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var initFlag bool
var KafkaRedisLog *gyredis.MyLog
var KafkaRedisConf *gyredis.RedisConf
var KafkaSender *gyredis.RedisClient

// KafkaSenderInit this kafka use local agent, it need redis
func KafkaSenderInit() error {
	KafkaRedisLog, _ = gyredis.NewMyLog("DEBUG", "../log/kafka_sender.log", 3, 1024*1024*100)
	KafkaRedisConf = gyredis.NewDefaultRedisConf()
	KafkaRedisConf.Addr = "127.0.0.1:6379"
	var err error
	KafkaSender, err = gyredis.NewRedisClient(KafkaRedisConf, KafkaRedisLog)
	if err != nil {
		KafkaRedisLog.Error("KafkaSenderInit failed",
			zap.String("config = ", KafkaRedisConf.Addr),
			zap.String("init error = ", err.Error()))
		return err
	}
	initFlag = true
	return nil
}

// SendSteps2KafkaLocal send local kafka client
func SendSteps2KafkaLocal(userSteps map[string]*metadata.UserSteps, userUnin map[string]string) error {
	if !initFlag {
		logger.Error("SendSteps2KafkaLocal error: local redis client not init yet...")
		return errors.NewUnknownError()
	}

	for oid, steps := range userSteps {
		unin := userUnin[oid]
		msg, err := generateStep(oid, unin, steps.Steps)
		if err != nil {
			logger.Error("generateStep error: %v", err)
			continue
		}
		if err = kafkaSendMsg(msg); err != nil {
			logger.Error("oid = %s, unin = %s, KafkaSendMsg error, err = %v", oid, unin, err)
			continue
		}
	}
	logger.Info("send total: %v step msg", len(userSteps))
	return nil
}

func generateStep(oid, unid string, steps map[string]int32) ([]byte, error) {
	var msg XGYProto.ReportMsg
	msg.MsgType = XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM.Enum()
	metrics := "metrics"
	msg.Metrics = &metrics
	topic := "yqz-update-step"
	msg.Topic = &topic
	msg.Key = &oid
	nt_date := util.GetLocalFormatTime()
	msg.Dt = &nt_date
	msg.Fields = make([]*XGYProto.KeyVal, 0)
	msg.Fields = addKeySValField(msg.Fields, "busi_type", []byte("YQZ_UpdateStep"))
	msg.Fields = addKeySValField(msg.Fields, "oid", []byte(oid))
	msg.Fields = addKeySValField(msg.Fields, "unid", []byte(unid))
	date_step_json, err := json.Marshal(steps)
	if err != nil {
		logger.Error("json marshal error, err = %v, date_step = %+v", err, steps)
		return nil, err
	}
	msg.Fields = addKeySValField(msg.Fields, "date_step", date_step_json)
	logger.Debug("msg = %+v", msg)
	return proto.Marshal(&msg)
}

func addKeySValField(fields []*XGYProto.KeyVal, key string, val []byte) []*XGYProto.KeyVal {
	var field_key_val XGYProto.KeyVal
	field_key_val.Key = &key
	field_key_val.Sval = val
	fields = append(fields, &field_key_val)
	return fields
}

func kafkaSendMsg(msg []byte) error {
	_, err := KafkaSender.LPush("gy_feeds_0", msg).Result()
	if err != nil {
		logger.Error("lpush error, err = %v, msg = %s", err, string(msg))
		return err
	}
	return nil
}
