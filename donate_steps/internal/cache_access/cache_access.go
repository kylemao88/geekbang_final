//
//

package cache_access

import (
	"strconv"
	"strings"
	"time"

	"git.code.oa.com/going/l5"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"go.uber.org/zap"
)

type RedisConfig struct {
	IpPorts  string `toml:"ip_ports"`
	ModId    int32  `toml:"mod_id"`
	CmdId    int32  `toml:"cmd_id"`
	PassWord string `toml:"pwd"`
}

var RedisLog *gyredis.MyLog
var RedisConf *gyredis.RedisConf
var RedisClient *gyredis.RedisClient

var cLoc *time.Location

func init() {
	cLoc, _ = time.LoadLocation("Asia/Shanghai")
}

func RedisInit(config RedisConfig) {
	RedisLog, _ = gyredis.NewMyLog("DEBUG", "../log/redis.log", 4, 10240000)
	RedisConf = gyredis.NewDefaultRedisConf()
	if config.PassWord != "" {
		RedisConf.Password = config.PassWord
	}
	if config.IpPorts != "" {
		RedisConf.Addr = config.IpPorts
	} else {
		ser, err := l5.ApiGetRouteTable(config.ModId, config.CmdId)
		if err != nil {
			log.Error("get redis server ip and port error, err = %s, modid = %d, cmdid = %d",
				err.Error(), config.ModId, config.CmdId)
			panic("init redis error")
		}
		ip_ports := ""
		for _, it := range ser {
			ip_ports += it.Ip + ":" + strconv.Itoa(it.Port) + ","
		}
		ip_ports = strings.TrimRight(ip_ports, ",")
		RedisConf.Addr = ip_ports
		log.Debug("redis ip_ports = %s", ip_ports)
	}
	var err error
	RedisClient, err = gyredis.NewRedisClient(RedisConf, RedisLog)
	if err != nil {
		RedisLog.Error("mylog.NewRedisClient failed",
			zap.String("config = ", RedisConf.Addr),
			zap.String("init error = ", err.Error()))
		panic("init redis error")
	}
}
