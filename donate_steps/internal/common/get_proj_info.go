//
//

package common

import (
	"context"
	"errors"
	"fmt"
	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/go_common_v2/util"
	"strconv"
)

type ExtDonateStruct struct {
	Def   []int  `json:"def"`
	Month []int  `json:"month"`
	Yqj   []int  `json:"yqj"`
	Qunt  string `json:"qunt"`
	Nm    string `json:"nm"`
	Prc   string `json:"prc"`
	Targ  string `json:"targ"`
}

type ProjInfoStruct struct {
	Pid        string          `json:"id"`
	PName      string          `json:"title"`
	Summary    string          `json:"summary"`
	Status     string          `json:"status"`
	ListImg    string          `json:"listImg"`
	ImgMobList []string        `json:"img_mob_list"`
	ExtDonate  ExtDonateStruct `json:"ext_donate"`
}

func QueryProjInfo(pid int, proj_info *ProjInfoStruct) error {
	key := fmt.Sprintf("proj_%d_V3", pid)
	ckvval, err := util.GetCkv(context.Background(), "users", key)
	if err != nil {
		log.Error("ckv key = %s, get proj info error = %s", key, err.Error())
		return errors.New("ckv get proj info error")
	}

	info_str := fmt.Sprintf("%s", string(ckvval))
	if err := json.Unmarshal(ckvval, proj_info); err != nil {
		log.Error("user json parse error, err = %s, json = %s", err.Error(), info_str)
		return errors.New("parse proj info json error")
	}

	// log.Debug("ckv query proj info successfully, ret = %v", *proj_info)
	return nil
}

func BatchQueryProjInfo(pids []int) (map[int]ProjInfoStruct, error) {
	infos := make(map[int]ProjInfoStruct)

	if len(pids) == 0 {
		log.Error("pids is empty error")
		return infos, errors.New("pids empty")
	}

	var mckv_info util.CkvInfo
	mckv_info.Section = "users"
	mckv_info.Keys = make([]string, 0)
	for _, pid := range pids {
		if 0 == pid {
			continue
		}
		key := fmt.Sprintf("proj_%d_V3", pid)
		mckv_info.Keys = append(mckv_info.Keys, key)
	}

	res, _ := util.GetMCkvVals(&mckv_info)
	if mckv_info.Err != nil {
		log.Error("GetMCkvVals error, err = %v", mckv_info.Err)
		return infos, mckv_info.Err
	}

	for k, v := range res {
		var project ProjInfoStruct
		info_str := fmt.Sprintf("%s", v.(string))
		err := json.Unmarshal([]byte(info_str), &project)
		if err != nil {
			log.Error("project json parse error, err = %v, k = %s, json = %s", err, k, info_str)
			return infos, err
		}
		// 项目id不能为0
		if 0 == len(project.Pid) || "0" == project.Pid {
			continue
		}
		pid, _ := strconv.Atoi(project.Pid)
		infos[pid] = project
	}

	return infos, nil
}
