//
//

package common

/*
#cgo CFLAGS: -I./wxlib/src -I./wxlib/lib/include64
#cgo LDFLAGS: -L./wxlib/src -lwxencrypt -L./wxlib/lib/lib64 -lssl -L./wxlib/lib/lib64 -lcrypto -lstdc++
#include "WXWapper.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
) //注意cgo的注释和 import "C"之间不能有空行！！！ 注意cgo的注释和 import "C"之间不能有空行！！！ 注意cgo的注释和 import "C"之间不能有空行！！！

type WxStepsData struct {
	Timestamp int64 `json:"timestamp"`
	Steps     int64 `json:"step"`
}

type WxStepsArray struct {
	Stepinfolist []WxStepsData `json:"stepInfoList"`
}

func DecodeWxData(appid string, session_key string, en_data string, iv_data string) (WxStepsArray, error) {
	var ret WxStepsArray
	var ret_size int
	var ret_code int

	c_appid := C.CString(appid)
	defer C.free(unsafe.Pointer(c_appid))
	c_session_key := C.CString(session_key)
	defer C.free(unsafe.Pointer(c_session_key))
	c_en_data := C.CString(en_data)
	defer C.free(unsafe.Pointer(c_en_data))
	c_iv_data := C.CString(iv_data)
	defer C.free(unsafe.Pointer(c_iv_data))

	decode_data := C.DecryptData(c_appid, c_session_key, c_en_data, c_iv_data,
		(*C.int)(unsafe.Pointer(&ret_size)), (*C.int)(unsafe.Pointer(&ret_code)))
	defer C.free(unsafe.Pointer(decode_data))

	if ret_code != 0 {
		return ret, fmt.Errorf("decode wx data error, code = %d", ret_code)
	}
	tmp_decode_data := []byte(C.GoStringN(decode_data, (C.int)(ret_size)))
	if err := json.Unmarshal(tmp_decode_data, &ret); err != nil {
		log.Error("user json parse error, err = %s, json = %s", err.Error(), string(tmp_decode_data))
		return ret, errors.New("parse wx session data json error")
	}

	return ret, nil
}

type WxGroupData struct {
	GroupId   string `json:"openGId"`
	MsgTicket string `json:"msgTicket"`
}

func DecodeWxEncodeData(appId, sessionKey, enData, ivData string) ([]byte, error) {
	var retSize, retCode int
	cAppId := C.CString(appId)
	defer C.free(unsafe.Pointer(cAppId))
	cSessionKey := C.CString(sessionKey)
	defer C.free(unsafe.Pointer(cSessionKey))
	cEnData := C.CString(enData)
	defer C.free(unsafe.Pointer(cEnData))
	cIvData := C.CString(ivData)
	defer C.free(unsafe.Pointer(cIvData))
	decodeData := C.DecryptData(cAppId, cSessionKey, cEnData, cIvData,
		(*C.int)(unsafe.Pointer(&retSize)),
		(*C.int)(unsafe.Pointer(&retCode)))
	defer C.free(unsafe.Pointer(decodeData))
	if retCode != 0 {
		log.Error("decode wx data error, code = %d", retCode)
		return nil, errors.New("decode wx data error")
	}

	return []byte(C.GoStringN(decodeData, (C.int)(retSize))), nil
}

func DecodeWxGroupId(appId, unionId, eData, eIv string) (string, string, error) {
	sKey, err := QueryWxSessionKey(appId, unionId)
	if err != nil {
		log.Error("QueryWxSessionKey error, err = %v, appId = %s, unionId = %s", err, appId, unionId)
		return "", "", err
	}

	data, err := DecodeWxEncodeData(appId, sKey, eData, eIv)
	if err != nil {
		log.Error("DecodeWxEncodeData error, err = %v, appId = %s, unionId = %s, eData = %s, eIv = %s",
			err, appId, sKey, eData, eIv)
		return "", "", err
	}

	log.Info("group data = %s", string(data))

	var wxGroupIData WxGroupData
	if err := json.Unmarshal(data, &wxGroupIData); err != nil {
		log.Error("user json parse error, err = %v, json = %s", err, string(data))
		return "", "", errors.New("parse wx session data json error")
	}
	return wxGroupIData.GroupId, wxGroupIData.MsgTicket, nil
}
