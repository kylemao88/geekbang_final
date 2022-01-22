create DATABASE gongyi_yqz;

// match tables

// 名词解释: activity表示活动, team表示小队, match_activity表示配捐活动, match_rules表示配捐规则
CREATE TABLE `gongyi_yqz`.`t_activity_info` (
    `f_activity_id` varchar(64) NOT NULL COMMENT "活动id",
    `f_match_event_id` varchar(64) DEFAULT "" COMMENT "配捐活动id, 空表示不支持配捐",
    `f_activity_creator` varchar(64) DEFAULT "" COMMENT "活动创建者id",
    `f_type` int(2) DEFAULT 0 COMMENT "0表示运营平台发起, 1表示手机端发起",
    `f_desc` varchar(256) DEFAULT "" COMMENT "活动描述",
    `f_slogan` varchar(256) DEFAULT "" COMMENT "活动口号",
    `f_org_name` varchar(256) DEFAULT "" COMMENT "团体名称, 适用于移动端发起, 区别企业名称, 企业名称通过配捐信息查询",
    `f_org_head` varchar(256) DEFAULT "" COMMENT "团体头图, 适用于移动端发起, 区别企业logo, 企业logo通过配捐信息查询",
    `f_route_id` varchar(64) DEFAULT "" COMMENT "地图id",
    `f_bgpic` varchar(256) DEFAULT "" COMMENT "活动自定义主题图片, 空表示云游中国地图（每周更新）",
    `f_bgpic_status` int(2) DEFAULT 0 COMMENT "自定义图片审核状态, 0表示未审核, 1表示审核通过, 2表示审核失败",
    `f_start_time` datetime NOT NULL COMMENT "活动开始时间",
    `f_end_time` datetime NOT NULL COMMENT "活动结束时间",
    `f_team_mode` int(2) DEFAULT 0 COMMENT "小队模式 0：活动有小队用户不必须加入队伍 1：活动有小队并且用户必须加入小队 2：活动无小队",
    `f_team_off` int(2) DEFAULT 0 COMMENT "0表示开放组队, 1表示关闭组队",
    `f_team_member_limit` int(11) DEFAULT 2000 COMMENT "小队人数上限",
    `f_show_sponsor` int(2) DEFAULT 0 COMMENT "是否展示发起人信息, 0表示不展示发起人信息, 1表示展示发起人信息",
    `f_match_off` int(2) DEFAULT 0 COMMENT "是否支持配捐 0表示支持配捐, 1表示不支持配捐, 默认活动支持配捐",
    `f_status` int(2) DEFAULT 0 COMMENT "0：未审核(手机端发起支付成功才审核通过)  1：活动未开始  2：活动进行中  3：活动已结束（时间过期） 4：活动停止（运营平台手动停止） 5：活动暂停（运营平台手动暂停）",
    `f_snapshot` int(2) DEFAULT 0 COMMENT "0：活动未快照  1：活动快照成功（活动结束后）",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_activity_id`),
    KEY `index_snapshot` (`f_snapshot`),
    KEY `index_start_time` (`f_start_time`),
    KEY `index_end_time` (`f_end_time`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "活动信息表";
alter table t_activity_info add f_rule varchar(3072) DEFAULT "" COMMENT "活动规则（pc端发起的活动，可以直接配置）";
alter table t_activity_info add f_company_id varchar(64) DEFAULT "" COMMENT "活动创建企业id(运营平台)";
alter table t_activity_info add f_white_type int(2) DEFAULT 0 COMMENT "白名单类型，0表示不需要，1表示要白名单";
alter table t_activity_info add f_cover varchar(256) DEFAULT "" COMMENT "活动封面图片地址";
alter table t_activity_info add f_relation_desc varchar(256) DEFAULT "" COMMENT "活动关联描述(把一些活动关联起来)";
alter table t_activity_info add f_forward_pic varchar(256) DEFAULT "" COMMENT "转发图片地址";
//
alter table t_activity_info add f_multi_match int(2) DEFAULT 0 COMMENT "是否有多个配捐，0表示没有，1表示有";
alter table t_activity_info add f_multi_match_id varchar(256) DEFAULT "" COMMENT "多个配捐id，f_multi_match = 1 有值";

// 对应doc的`t_event_info`表
CREATE TABLE `gongyi_yqz`.`t_team` (
    `f_activity_id` varchar(64) NOT NULL COMMENT "活动id",
    `f_team_id` varchar(64) NOT NULL COMMENT "小队id",
    `f_team_type` int(11) DEFAULT 1 COMMENT "1 表示普通小队 2 表示私密小队，非队员不可围观",
    `f_team_desc` varchar(256) NOT NULL COMMENT "小队名称",
    `f_team_creator` varchar(64) DEFAULT "admin" COMMENT "小队创建者id, 默认admin表示为运营平台创建",
    `f_team_leader` varchar(64) DEFAULT "" COMMENT "活动队长id",
    `f_team_flag` int(2) DEFAULT 0 COMMENT "小队标签, 0表示用户创建的小队, 1表示运营平台系统创建的默认小队",
    `f_status` int(2) DEFAULT 1 COMMENT "1 正常 2 解散(系统默认小队不能解散)",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (f_activity_id,`f_team_id`),
    KEY `index_modify` (`f_modify_time`),
    KEY `index_status`(`f_status`),
    KEY `index_create` (`f_create_time`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "活动小队表";

CREATE TABLE `gongyi_yqz`.`t_activity_member` (
    `f_activity_id` varchar(64) NOT NULL COMMENT "活动id",
    `f_user_id` varchar(64) NOT NULL COMMENT "活动成员id",
    `f_status` int(2) DEFAULT 0 COMMENT "0 表示加入活动(兼容以前的数据)，1 表示退出活动，2 表示黑名单",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_activity_id`,`f_user_id`),
    KEY `index_user_id` (`f_user_id`),
	KEY `index_create` (`f_create_time`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "活动成员表";
alter table t_activity_member add f_arrive_time varchar(64) DEFAULT "" COMMENT "到达终点时间";

// 对应doc的`t_event_week_stat`表
CREATE TABLE `gongyi_yqz`.`t_team_member` (
    `f_activity_id` varchar(64) NOT NULL COMMENT "活动id",
    `f_team_id` varchar(64) NOT NULL COMMENT "小队id",
    `f_user_id` varchar(64) NOT NULL COMMENT "小队成员id",
    `f_in_team` int(2) DEFAULT 2 COMMENT "1 表示未参与活动，2 表示参与活动",
    `f_status` int(2) DEFAULT 0 COMMENT "状态, 预留",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_activity_id`,`f_team_id`,`f_user_id`),
    KEY `index_user_id` (`f_user_id`),
	KEY `index_create` (`f_create_time`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "小队成员表";

// 小队统计周期历史数据表
// TODO: 后续这个历史数据大表要考虑迁移到TDSQL
CREATE TABLE `gongyi_yqz`.`t_team_history` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_activity_id` varchar(64) NOT NULL COMMENT "活动id",
    `f_team_id` varchar(64) NOT NULL COMMENT "小队id",
    `f_team_steps` BIGINT(20) DEFAULT 0 COMMENT "小队周期总步数",
    `f_team_rank` int(11) DEFAULT 0 COMMENT "小队周期排名",
    `f_status` int(2) DEFAULT 0 COMMENT "状态, 预留",
    `f_start_time` datetime NOT NULL COMMENT "小队周期开始时间",
    `f_end_time` datetime NOT NULL COMMENT "小队周期结束时间",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_id`),
    UNIQUE KEY `index_team` (`f_activity_id`,`f_team_id`),
    KEY `index_team_rank` (`f_team_rank`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "小队历史表";

// 小队成员周期的历史数据表
// TODO: 后续这个历史数据大表要考虑迁移到TDSQL
CREATE TABLE `gongyi_yqz`.`t_team_member_history` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_activity_id` varchar(64) NOT NULL COMMENT "活动id",
    `f_team_id` varchar(64) NOT NULL COMMENT "小队id",
    `f_user_id` varchar(64) NOT NULL COMMENT "小队成员id",
    `f_user_steps` BIGINT(20) DEFAULT 0 COMMENT "成员周期总步数",
    `f_user_rank` int(11) DEFAULT 0 COMMENT "成员周期排名",
    `f_status` int(2) DEFAULT 0 COMMENT "状态, 预留",
    `f_in_team` int(2) DEFAULT 2 COMMENT "1 表示未参与活动，2 表示参与活动",
    `f_start_time` datetime NOT NULL COMMENT "周期开始时间",
    `f_end_time` datetime NOT NULL COMMENT "周期结束时间",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_id`),
    UNIQUE KEY `index_user` (`f_activity_id`,`f_team_id`,`f_user_id`),
    KEY `index_user_rank` (`f_user_rank`),
    KEY `index_in_team` (`f_in_team`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "小队成员历史表";

CREATE TABLE `gongyi_yqz`.`t_activity_member_history` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_activity_id` varchar(64) NOT NULL COMMENT "活动id",
    `f_user_id` varchar(64) NOT NULL COMMENT "活动成员id",
    `f_user_steps` BIGINT(20) DEFAULT 0 COMMENT "成员周期总步数",
    `f_activity_rank` int(11) DEFAULT 0 COMMENT "活动周期成员排名",
    `f_status` int(2) DEFAULT 0 COMMENT "状态, 预留",
    `f_in_activity` int(2) DEFAULT 2 COMMENT "1 表示未参与活动，2 表示参与活动",
    `f_start_time` datetime NOT NULL COMMENT "活动周期开始时间",
    `f_end_time` datetime NOT NULL COMMENT "活动周期结束时间",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_id`),
    UNIQUE KEY (`f_activity_id`,`f_user_id`),
    KEY `index_user_id` (`f_user_id`),
    KEY `index_in_activity` (`f_in_activity`),
    KEY `index_user_rank` (`f_activity_rank`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "活动成员历史表";

CREATE TABLE `gongyi_yqz`.`t_user_match_summary` (
	`f_user_id` varchar(64) NOT NULL COMMENT "操作者oid",
    `f_total_match_fund` bigint NOT NULL COMMENT "配捐金额",
    `f_total_match_time` int(32) NOT NULL COMMENT "配捐次数",
    `f_total_match_day` int(32) NOT NULL COMMENT "配捐天数",
    `f_match_date` datetime NOT NULL COMMENT "最后配捐日期",
	`f_status` int(2) DEFAULT 0 COMMENT "",
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
	PRIMARY KEY (`f_user_id`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "用户总配捐统计";

CREATE TABLE `gongyi_yqz`.`t_user_match_status` (
	`f_user_id` varchar(64) NOT NULL COMMENT "操作者oid",
	`f_activity_id` varchar(64) NOT NULL DEFAULT "" COMMENT "活动id",
    `f_total_match_fund` bigint NOT NULL COMMENT "配捐金额",
    `f_total_match_step` bigint NOT NULL COMMENT "配捐步数",
    `f_total_match_time` int(32) NOT NULL COMMENT "配捐次数",
    `f_match_combo` int(32) NOT NULL COMMENT "配捐连击",
	`f_status` int(2) DEFAULT 0 COMMENT "",
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
	PRIMARY KEY (`f_user_id`,`f_activity_id`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "用户活动配捐统计";

CREATE TABLE `gongyi_yqz`.`t_user_match_record` (
    `f_id` bigint(29) NOT NULL AUTO_INCREMENT,
	`f_match_id` varchar(64) NOT NULL COMMENT "配捐流水号",
	`f_user_id` varchar(64) NOT NULL COMMENT "操作者oid",
	`f_match_item_id` varchar(64) NOT NULL COMMENT "配捐活动id",
	`f_match_org_id` varchar(64) NOT NULL COMMENT "配捐企业id",
	`f_activity_id` varchar(64) NOT NULL DEFAULT "" COMMENT "活动id",
    `f_match_fund` int(32) NOT NULL COMMENT "配捐金额",
    `f_match_step` int(32) NOT NULL COMMENT "配捐步数",
    `f_match_combo` int(32) NOT NULL COMMENT "配捐连击",
    `f_match_date` datetime NOT NULL COMMENT "配捐时间戳",
    `f_date` datetime NOT NULL DEFAULT NOW() COMMENT "配捐日期",
    `f_op_type` int(2) DEFAULT 0 COMMENT "行为类型 1:普通配捐 2:额外配捐",
	`f_status` int(2) DEFAULT 0 COMMENT "",
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
    `f_coupons_flag` bool DEFAULT false COMMENT "是否伴随消费券",
	PRIMARY KEY (`f_id`),
	UNIQUE KEY `index_match_id` (`f_match_id`),
	UNIQUE KEY `match_record_date` (`f_user_id`,`f_activity_id`,`f_date`),
    KEY `match_time` (`f_create_time`)
    KEY `match_date` (`f_match_date`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "配捐流水记录";

CREATE TABLE `gongyi_yqz`.`t_match_activity` (
	`f_event_id` varchar(64) NOT NULL COMMENT "配捐活动id",
	`f_company_id` varchar(64) NOT NULL COMMENT "企业id",
	`f_pid` varchar(64) NOT NULL COMMENT "配捐项目id",
	`f_rule_id` varchar(64) NOT NULL COMMENT "配捐规则id",
    `f_target_fund` bigint(20) DEFAULT 0 COMMENT "配捐目标总金额, 单位分",
    `f_remind_fund` bigint(20) DEFAULT 100000 COMMENT "配捐活动消息提醒剩余金额, 单位分",
    `f_cost_fund` bigint(20) DEFAULT 0 COMMENT "配捐已花费金额",
    `f_cost_cnt` bigint(20) DEFAULT 0 COMMENT "配捐人次",
    `f_steps` bigint(20) DEFAULT 0 COMMENT "配捐总步数",
    `f_match_type` int(2) DEFAULT 0 COMMENT "配捐活动类型 1:常规配捐 2:企业配捐",
    `f_match_mode` int(2) DEFAULT 0 COMMENT "配捐模式 0表示平均配捐（小池子每日上限） 1表示累计配捐（大池子）",
    `f_anti_black` varchar(1024) DEFAULT '' COMMENT "防刷规则, json 格式",
	`f_start_time` datetime NOT NULL,
	`f_end_time` datetime NOT NULL,
	`f_status` int(2) DEFAULT 0 COMMENT "0：配捐未开始  1：配捐进行中  2：配捐完成（钱花完） 3：配捐完成（没花完过期了）",
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
    `f_coupons_flag` int(2) DEFAULT 0 COMMENT "是否支持消费券",
    PRIMARY KEY (`f_event_id`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "配捐活动记录";

CREATE TABLE `gongyi_yqz`.`t_match_rules` (
    `f_rule_id` varchar(64) NOT NULL DEFAULT "default" COMMENT "配捐规则id",
	`f_basic_fund` int(2) DEFAULT 0 COMMENT "基础配捐金额",
	`f_basic_wave` int(2) DEFAULT 0 COMMENT "基础配捐波动",
	`f_buff_fund` int(2) DEFAULT 0 COMMENT "buff配捐金额",
	`f_buff_wave` int(2) DEFAULT 0 COMMENT "buff配捐波动",
	`f_buff_threshold` int(2) DEFAULT 0 COMMENT "触发buff的连续配捐次数",
	`f_buff_percent` int(2) DEFAULT 0 COMMENT "触发buff配捐比例",
    `f_match_quota` int(2) DEFAULT 0 COMMENT "配捐的单份额度",
    `f_type` int(2) NOT NULL DEFAULT 0 COMMENT "规则类型: 1:固定兑换(只能兑一份) 2:按比兑换",
	`f_status` int(2) DEFAULT 0 COMMENT "",
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
	`f_min_step` int(2) DEFAULT 0 COMMENT "最小步数",
	`f_max_step` int(2) DEFAULT 0 COMMENT "最大步数",
	`f_min_match` int(2) DEFAULT 0 COMMENT "最小配捐",
	`f_max_match` int(2) DEFAULT 0 COMMENT "最大配捐",
	PRIMARY KEY (`f_rule_id`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "配捐规则";

CREATE TABLE `gongyi_yqz`.`t_step` (
	`f_user_id` varchar(64) NOT NULL,
	`f_step` int(20) DEFAULT 0,
	`f_update_method` int(2) NOT NULL COMMENT "1 auto update 2 initial update",
	`f_date` date NOT NULL,
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
	PRIMARY KEY (`f_user_id`,`f_date`),
	KEY `index_date` (`f_date`),
	KEY `index_create` (`f_create_time`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "每日步数表";

CREATE TABLE `gongyi_yqz`.`t_step_history` (
	`f_user_id` varchar(64) NOT NULL,
	`f_step` int(20) DEFAULT 0,
	`f_update_method` int(2) NOT NULL COMMENT "1 auto update 2 initial update",
	`f_date` date NOT NULL,
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
	PRIMARY KEY (`f_user_id`,`f_date`),
	KEY `index_date` (`f_date`),
	KEY `index_create` (`f_create_time`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "每日步数历史表";

CREATE TABLE `gongyi_yqz`.`t_user` (
    `f_user_id` varchar(64) NOT NULL,
    `f_appid` varchar(64) NOT NULL,
    `f_unin_id` varchar(64) NOT NULL,
    `f_join_event` int(2) NOT NULL COMMENT "0 表示未设置，1 表示没有加入活动，2表示加入活动",
    `f_backend_update` int(2) NOT NULL COMMENT "是否支持后台更新， 1 表示不支持，2 表示支持",
    `f_status` int(2) NOT NULL COMMENT "0 表示未捐过步，1表示捐过步",
    `f_sub_time` datetime NOT NULL COMMENT "绑定时间",
    `f_create_time` datetime NOT NULL,
    `f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_user_id`),
    KEY `index_unin_id`(`f_unin_id`),
    KEY `index_backend_update` (`f_join_event`,`f_backend_update`,`f_status`),
    KEY `index_create` (`f_create_time`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "个人信息表";
alter table t_user add f_phone_num varchar(64) DEFAULT "" COMMENT "用户手机号码";

CREATE TABLE `gongyi_yqz`.`t_company_match_stat` (
	`f_company_id` varchar(64) NOT NULL COMMENT "企业id",
    `f_match_fund` bigint(20) DEFAULT 0 COMMENT "总金额, 单位分",
    `f_match_cnt` bigint(20) DEFAULT 0 COMMENT "总次数",
	`f_stat_date` datetime NOT NULL,
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_company_id`,`f_stat_date`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "机构每日配捐记录";

CREATE TABLE `gongyi_yqz`.`t_white_list` (
	`f_activity_id` varchar(64) NOT NULL DEFAULT "" COMMENT "活动id",
    `f_phone_num` varchar(64) DEFAULT "" COMMENT "手机号码",
    `f_status` int(2) DEFAULT 0 COMMENT "",
	`f_create_time` datetime NOT NULL,
	`f_modify_time` datetime NOT NULL,
    PRIMARY KEY (`f_activity_id`,`f_phone_num`)
) ENGINE = InnoDB CHARSET = utf8mb4 COMMENT "活动型用户白名单";
