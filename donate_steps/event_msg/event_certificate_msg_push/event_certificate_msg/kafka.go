package event_certificate_msg

import (
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/gongyi_base/component/gy_kafka"
	"git.code.oa.com/gongyi/gongyi_base/proto/gy_agent"
	"google.golang.org/protobuf/proto"
	"time"
)

type Client struct {
	SyncCtx **gy_kafka.ConsumerCtx
}

func (client *Client) DealMsg(Msg *gy_kafka.Message, rnum int) bool {
	log.Info("%v, %s dd = %v client Message claimed: key = %s,  topic = %s, partition = %v, offset = %v",
		time.Now(), Msg.MemberID, Msg.GenerationID, string(Msg.Msg.Key), Msg.Msg.Topic, Msg.Msg.Partition, Msg.Msg.Offset)

	msg := &XGYProto.ReportMsg{}
	err := proto.Unmarshal(Msg.Msg.Value, msg)
	if err != nil {
		log.Error("proto decode error[%s]\n", err.Error())
		return true
	}
	log.Info("msg = %+v, topic = %s", msg, msg.GetTopic())

	if msg.Key == nil {
		log.Error("msg have no key")
		return true
	}

	if msg.MsgType == nil {
		log.Error("no msg type")
		return true
	}

	if msg.GetMsgType() != XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM {
		log.Error("msg type error")
		return true
	}

	busi_type := ""
	for _, values := range msg.GetFields() {
		if values.GetKey() == "busi_type" {
			busi_type = string(values.GetSval())
		}
	}

	if busi_type == "YQZ_WeekChallenge" {
		wc, err := parseWeekChallenge(msg)
		if err != nil {
			log.Error("parseWeekChallenge error, err = %v, msg = %+v", err, msg)
			return true
		}

		log.Debug("WeekChallenge = %+v", wc)

		if len(wc.Oid) == 0 {
			log.Info("empty data = %+v", wc)
			return true
		}

		err = EventCertificateMsgPush(wc)
		if err != nil {
			log.Error("EventCertificateMsgPush error, err = %v, wc = %v", err, wc)
		}
	}
	return true
}

func parseWeekChallenge(msg *XGYProto.ReportMsg) (common.WeekChallenge, error) {
	var ret common.WeekChallenge
	for _, values := range msg.GetFields() {
		switch values.GetKey() {
		case "oid":
			ret.Oid = string(values.GetSval())
		case "step":
			ret.Step = int64(values.GetIval())
		case "rank":
			ret.Rank = int64(values.GetIval())
		case "num":
			ret.Num = int64(values.GetIval())
		case "time":
			ret.Time = string(values.GetSval())
		}
	}
	return ret, nil
}
