//

package alarm

import (
	"git.code.oa.com/gongyi/gy_warn_msg_httpd/msgclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
)

type AlarmMsgConfig struct {
	Busi     string `yaml:"busi"`
	Sender   string `yaml:"sender"`
	Receiver string `yaml:"receiver"`
	Septime  int32  `yaml:"septime"`
}

// CallAlarmFunc use keyou support client
func CallAlarmFunc(msg string) {
	logger.Warnf("%v - msg: %v", util.GetCallee(), msg)
	err := msgclient.Send_msg(msgclient.WXWARN|msgclient.RTX, "%s", msg)
	if err != nil {
		logger.Error("alarm error: %v", err)
	}
}

func CallSpecAlarmFunc(alarmType int, msg string) {
	logger.Warnf("%v - msg: %v", util.GetCallee(), msg)
	err := msgclient.Send_msg(alarmType, "%s", msg)
	if err != nil {
		logger.Error("alarm error: %v", err)
	}
}
