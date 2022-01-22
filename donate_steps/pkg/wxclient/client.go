//

package wxclient

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"git.code.oa.com/going/l5"
	"git.code.oa.com/gongyi/agw/log"
)

const TIMEOUT = 60
const DIAL_TIMEOUT = 10
const TLS_TIMEOUT = 10
const CONTINUE_TIMEOUT = 10
const IDLE_CNT = 200
const IDLE_TIMEOUT = 60
const IDLE_PER_HOST = 50

type WXClientErr int32

const (
	WX_SUCCESS WXClientErr = iota
	WX_FAILED
	WX_NO_PERM // no auth, need auth again
)

func (e WXClientErr) String() string {
	switch e {
	case WX_SUCCESS:
		return "WX_SUCCESS"
	case WX_FAILED:
		return "WX_FAILED"
	case WX_NO_PERM:
		return "WX_NO_PERM"
	default:
		return "UNKNOWN"
	}
}

// wxClientConfig opts...
type wxClientConfig struct {
	// for init fetch and decode
	ModID int32
	CmdID int32
	// For H5 Step handle
	ModIDH5   int32
	CmdIDH5   int32
	AppID     string
	AppSecret string
}

var Loc *time.Location

func init() {
	Loc, _ = time.LoadLocation("Asia/Shanghai")
}

// funcOption wraps a function that modifies wxClientConfig into an
// implementation of the WxClientOptions interface.
type funcOption struct {
	f func(*wxClientConfig)
}

func (fdo *funcOption) apply(do *wxClientConfig) {
	fdo.f(do)
}

func newFuncOption(f func(*wxClientConfig)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// WxClientOptions configures.
type WxClientOptions interface {
	apply(*wxClientConfig)
}

func WithL5(modID, cmdID int32) WxClientOptions {
	return newFuncOption(func(o *wxClientConfig) {
		o.ModID = modID
		o.CmdID = cmdID
	})
}

func WithH5Step(modID, cmdID int32, appID, appSecret string) WxClientOptions {
	return newFuncOption(func(o *wxClientConfig) {
		o.ModIDH5 = modID
		o.CmdIDH5 = cmdID
		o.AppID = appID
		o.AppSecret = appSecret
	})
}

var (
	client    *wxClient
	userToken sync.Map
	l5History sync.Map
)

type wxClient struct {
	client *http.Client
	config wxClientConfig
}

func InitWXClient(opts ...WxClientOptions) error {
	var HTTPTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: DIAL_TIMEOUT * time.Second,
			// KeepAlive: 30 * time.Second,
		}).DialContext, // connect params
		IdleConnTimeout:       IDLE_TIMEOUT * time.Second,
		ExpectContinueTimeout: CONTINUE_TIMEOUT * time.Second,
		TLSHandshakeTimeout:   TLS_TIMEOUT * time.Second,
		MaxIdleConns:          IDLE_CNT,
		MaxIdleConnsPerHost:   IDLE_PER_HOST,
	}
	// init client
	client = &wxClient{
		client: &http.Client{
			Transport: HTTPTransport,
			Timeout:   TIMEOUT * time.Second},
	}
	// apply all option
	for _, v := range opts {
		v.apply(&client.config)
	}
	return nil
}

func CloseWXClient() {

}

func IsWXTypeError(err error, errType WXClientErr) bool {
	return strings.Contains(err.Error(), errType.String())
}

func getWxL5(modID, cmdID int32) (*l5.Server, error) {
	// get wx server rs first
	ser, err := l5.ApiGetRoute(modID, cmdID)
	if err != nil {
		log.Error("get wx steps ip and port error, err = %s, modid = %d, cmdid = %d",
			err.Error(), modID, cmdID)
		l5Key := fmt.Sprintf("%v:%v", modID, cmdID)
		//if ser, ok := l5History[l5Key]; !ok {
		if ser, ok := l5History.Load(l5Key); !ok {
			return nil, err
		} else {
			log.Info("user rotten l5 addr: %v", ser)
			return ser.(*l5.Server), nil
		}
	}
	l5Key := fmt.Sprintf("%v:%v", modID, cmdID)
	l5History.Store(l5Key, ser)
	return ser, nil
}
