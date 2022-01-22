//
package mqclient

import (
	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	XGYProto "git.code.oa.com/gongyi/gongyi_base/proto/gy_agent"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"go.uber.org/zap"
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

func AddKeySValField(fields []*XGYProto.KeyVal, key string, val []byte) []*XGYProto.KeyVal {
	var field_key_val XGYProto.KeyVal
	field_key_val.Key = &key
	field_key_val.Sval = val
	fields = append(fields, &field_key_val)
	return fields
}

func AddKeyIntValField(fields []*XGYProto.KeyVal, key string, val float64) []*XGYProto.KeyVal {
	var field_key_val XGYProto.KeyVal
	field_key_val.Key = &key
	field_key_val.Ival = &val
	fields = append(fields, &field_key_val)
	return fields
}

func KafkaSendMsg(msg []byte) error {
	if !initFlag {
		logger.Error("SendSteps2KafkaLocal error: local redis client not init yet...")
		return errors.NewUnknownError()
	}

	_, err := KafkaSender.LPush("gy_feeds_0", msg).Result()
	if err != nil {
		logger.Error("lpush error, err = %v, msg = %s", err, string(msg))
		return err
	}
	return nil
}
