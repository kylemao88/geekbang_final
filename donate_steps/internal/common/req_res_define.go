//
//

package common

import "fmt"

// 修改活动请求和返回
type ModifyEventInfoReq struct {
	Oid     string `json:"oid"`
	EventId string `json:"event_id"`
	//EventStartTime  string `json:"event_start_time"`
	//EventEndTime    string `json:"event_end_time"`
	EventName string `json:"event_name"`
	EventType int    `json:"event_type"`
	//EventDesc       string `json:"event_desc"`
	//EventBgPic      string `json:"event_bg_pic"`
	//EventHeadPic    string `json:"event_head_pic"`
	//EventRuleId     int    `json:"event_rule_id"`
	//EventStatus     int    `json:"event_status"`
	//EventCreateOrgn string `json:"creater_orgn"`
}

type ModifyEventInfoRes struct {
	EventId string `json:"event_id"`
}

// 日榜请求和返回
type GetDayRankReq struct {
	EventId string `json:"event_id"`
	Oid     string `json:"oid"`
	Date    string `json:"date"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type GetDayRankRes struct {
	EventId        string     `json:"event_id"`
	Page           int        `json:"page"`
	Size           int        `json:"size"`
	TotalDonations int64      `json:"total_donations"`
	TotalSteps     int64      `json:"total_steps"`
	TotalMember    int64      `json:"total_member"`
	Data           []RankInfo `json:"data"`
}

// 本关榜
type GetLevelRankReq struct {
	EventId string `json:"event_id"`
	Oid     string `json:"oid"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type LevelRank struct {
	Oid          string `json:"oid"`
	Nick         string `json:"nick"`
	Head         string `json:"head"`
	Steps        int64  `json:"steps"`
	IsSelf       bool   `json:"is_self"`
	IsCreater    bool   `json:"is_creater"`
	RechargeFlag bool   `json:"recharge_flag"`
	Like         int64  `json:"like"`
	Progress     int64  `json:"progress"` // 前进进度
}

type GetLevelRankRes struct {
	EventId     string      `json:"event_id"`
	Page        int         `json:"page"`
	Size        int         `json:"size"`
	TotalMember int64       `json:"total_member"`
	RouteName   string      `json:"route_name"`
	Level       int         `json:"level"`
	FinishNum   int         `json:"finish_num"`
	Data        []LevelRank `json:"data"`
}

type RankInfo struct {
	Oid            string `json:"oid"`
	Nick           string `json:"nick"`
	Head           string `json:"head"`
	Steps          int64  `json:"steps"`
	DonateMoney    int64  `json:"donate_money"`
	RechargeFlag   bool   `json:"recharge_flag"`
	TotalFund      int64  `json:"total_fund"`
	TotalFundTimes int64  `json:"total_fund_times"`
	IsCreater      bool   `json:"is_creater"`
	IsSelf         bool   `json:"is_self"`
	Like           int64  `json:"like"`
}

// event rank req res
type GetEventRankReq struct {
	EventId string `json:"event_id"`
	Oid     string `json:"oid"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type GetEventRankRes struct {
	Page           int        `json:"page"`
	Size           int        `json:"size"`
	TotalSteps     int        `json:"total_steps"`
	TotalDonations int        `json:"total_donations"`
	TotalMoney     int        `json:"total_money"`
	TotalCount     int64      `json:"total_count"`
	Data           []RankInfo `json:"data"`
}

// 总榜请求和返回
type GetOverallRankReq struct {
	EventId string `json:"event_id"`
	Oid     string `json:"oid"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type GetOverallRankRes struct {
	Page            int        `json:"page"`
	Size            int        `json:"size"`
	TotalSteps      int64      `json:"total_steps"`
	TotalDonations  int64      `json:"total_donations"`
	TotalFund       int64      `json:"total_fund"`
	TotalRDonations int64      `json:"total_rdonations"`
	TotalRFund      int64      `json:"total_rfund"`
	TotalMember     int        `json:"total_member"`
	TotalLike       int64      `json:"total_like"`
	Data            []RankInfo `json:"data"`
}

// 	金主榜请求和返回
type GetRechargeRankReq struct {
	EventId string `json:"event_id"`
	Oid     string `json:"oid"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type RechargeRank struct {
	Oid            string `json:"oid"`
	Nick           string `json:"nick"`
	Head           string `json:"head"`
	RechargeFlag   bool   `json:"recharge_flag"`
	TotalFund      int64  `json:"total_fund"`
	TotalFundTimes int64  `json:"total_fund_times"`
	IsSelf         bool   `json:"is_self"`
	IsTeamLeader   bool   `json:"is_team_leader"`
}

type GetRechargeRankRes struct {
	EventId              string         `json:"event_id"`
	TotalFund            int64          `json:"total_fund"`
	TotalRFund           int64          `json:"total_rfund"`
	TotalDonations       int64          `json:"total_donations"`
	TotalRDonations      int64          `json:"total_rdonations"`
	TotalDonationTimes   int64          `json:"total_donation_times"`
	TotalRechargeMembers int            `json:"total_rehcarge_members"`
	Page                 int            `json:"page"`
	Size                 int            `json:"size"`
	Data                 []RechargeRank `json:"data"`
}

// 查询用户步数
type GetUserStepsReq struct {
	EventId      string `json:"event_id"`
	Oid          string `json:"oid"`
	Appid        string `json:"appid"`
	Code         string `json:"code"`
	EcryptedData string `json:"ecrypted_data"`
	EcryptedIv   string `json:"ecrypted_iv"`
	UniId        string `json:"uni_id"`
}

type UserDateSteps struct {
	Date  string `json:"date"`
	Steps int64  `json:"steps"`
}

type GetUserStepsRes struct {
	Oid                 string          `json:"oid"`
	Nick                string          `json:"nick"`
	Head                string          `json:"head"`
	Authorize           bool            `json:"authorize"`
	YesterdayDeltaSteps int64           `json:"yesterday_delta_steps"`
	TodayDeltaSteps     int64           `json:"today_delta_steps"`
	UserSteps           []UserDateSteps `json:"user_steps"`
}

// 捐步请求和返回
type DonateStepsH5Req struct {
	Oid     string `json:"oid"`
	Code    string `json:"code"`
	NotJoin bool   `json:"not_join"`
}

type DonateStepsH5Res struct {
	Oid       string `json:"oid"`
	Nick      string `json:"nick"`
	Head      string `json:"head"`
	TodayStep int64  `json:"today_step"`
}

// 捐步请求和返回
type DonateStepsReq struct {
	EventId      string `json:"event_id"`
	Oid          string `json:"oid"`
	Appid        string `json:"appid"`
	EcryptedData string `json:"ecrypted_data"`
	EcryptedIv   string `json:"ecrypted_iv"`
	UniId        string `json:"uni_id"`
	NotJoin      bool   `json:"not_join"`
}

type DonateStepsRes struct {
	Oid       string `json:"oid"`
	Nick      string `json:"nick"`
	Head      string `json:"head"`
	WeekId    string `json:"week_id"`
	WeekStep  int64  `json:"week_step"`
	TodayStep int64  `json:"today_step"`
}

// 自动捐步授权和查询请求和返回
type AutoDonateStepsReq struct {
	MOid  string `json:"moid"`
	Oid   string `json:"oid"`
	Appid string `json:"appid"`
	Type  int    `json:"type"` // 1 表示解除绑定，2 表示绑定
}

type AutoDonateStepsRes struct {
	SubBeginTime    string `json:"sub_begin_time"`
	SubEndTime      string `json:"sub_end_time"`
	AutoBangDayTime string `json:"auto_bang_day_time"`
}

type AutoDonateStepsReqV2 struct {
	Oid   string `json:"oid"`
	UniId string `json:"moid"`
	AppId string `json:"appid"`
	Type  int    `json:"type"` // 1 表示解除绑定，2 表示绑定，3 表示强制刷新授权时间
}

type AutoDonateStepsResV2 struct {
	SubBeginTime    string `json:"sub_begin_time"`
	SubEndTime      string `json:"sub_end_time"`
	AutoBangDayTime string `json:"auto_bang_day_time"`
}

// 查询是否授权请求和返回
type GetAutoDonateInfoReq struct {
	Oid string `json:"oid"`
}

type GetAutoDonateInfRes struct {
	Exist        bool   `json:"exist"`  // 是否绑定过
	Status       int    `json:"status"` // 绑定状态
	SubBeginTime string `json:"sub_begin_time"`
	SubEndTime   string `json:"sub_end_time"`
}

// 加入活动请求返回
type JoinEventReq struct {
	Oid     string `json:"oid"`
	EventId string `json:"event_id"`
	Type    int    `json:"type"` // 1 表示加入，2 表示退出
	Week    string `json:"week"`
}

type JoinEventRes struct {
	EventId string `json:"event_id"`
}

// 查询路线信息
type GetLineInfoReq struct {
	Oid     string `json:"oid"`
	EventId string `json:"event_id"`
}

type RouterData struct {
	Day            int     `json:"day"`
	Id             int     `json:"id"`
	Level          int     `json:"level"`
	Name           string  `json:"name"`
	Distance       float32 `json:"distance"`
	Dsteps         int64   `json:"d_steps"`
	TravelDistance float64 `json:"travel_distance"`
}

type Level struct {
	Id          int     `json:"id"`
	Level       int     `json:"level"`
	Name        string  `json:"name"`
	Distance    float32 `json:"distance"`
	Thumbnail   string  `json:"thumbnail"`
	Status      int     `json:"status"`
	AttriveDate string  `json:"attrive_date"`
}

type GetLineInfoRes struct {
	RouterInfo RouterData  `json:"router_info"`
	LevelList  []Level     `json:"level_list"`
	Friends    []MNickHead `json:"friends"`
	FinishFlag bool        `json:"finish_flag"`
}

// 查询每日队伍捐赠的总步数
type GetLineDayStepsReq struct {
	EventId   string `json:"event_id"`
	Oid       string `json:"oid"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type DaySteps struct {
	Date      string  `json:"date"`
	Steps     int64   `json:"steps"`
	Kilometer float32 `json:"kilometer"`
}

type GetLineDayStepsRes struct {
	EventId       string     `json:"event_id"`
	TotalKilomter float32    `json:"total_kilomter"`
	DateDaySteps  []DaySteps `json:"date_day_steps"`
}

type GetEventCommentReq struct {
	EventId string `json:"event_id"`
	Oid     string `json:"oid"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	// --story=867397871
	ActivityID string `json:"activity_id"`
}

type ReplyInfo struct {
	IsSelf    bool   `json:"is_self"`
	Rid       string `json:"rid"`
	Oid       string `json:"oid"`
	Like      int64  `json:"like"`
	Nick      string `json:"nick"`
	Reply     string `json:"reply"`
	ReplyTime int64  `json:"reply_time"`
	// --story=867397871
	//ToOid    string `json:"to_oid"`
	ToNick   string `json:"to_nick"`
	Activity string `json:"activity"`
}

type CommentInfo struct {
	CommentId   string      `json:"comment_id"`
	Like        int64       `json:"like"`
	Top         bool        `json:"top"`
	Pid         int         `json:"pid"`
	Title       string      `json:"title"`
	Nick        string      `json:"nick"`
	Head        string      `json:"head"`
	Comment     string      `json:"comment"`
	CommentTime int64       `json:"comment_time"`
	Type        int         `json:"type"`
	ShowType    int         `json:"show_type"`
	ReplyCount  int         `json:"reply_count"`
	ReplyList   []ReplyInfo `json:"reply_list"`
	// --story=867397871
	Activity string `json:"activity"`
	Oid      string `json:"oid"`
}

type GetEventCommentRes struct {
	Count int           `json:"count"`
	List  []CommentInfo `json:"list"`
}

// 查询回复评论
type GetReplyCommentReq struct {
	CommentId string `json:"comment_id"`
	Oid       string `json:"oid"`
	Page      int    `json:"page"`
	Size      int    `json:"size"`
	// --story=867397871
	Activity string `json:"activity_id"`
}

type GetReplyCommentRes struct {
	Count int         `json:"count"`
	List  []ReplyInfo `json:"list"`
}

// 活动评论
type CommentEventReq struct {
	EventId string `json:"event_id"`
	Oid     string `json:"oid"`
	Comment string `json:"comment"`
	// --story=867397871
	ActivityID string `json:"activity_id"`
}

type CommentEventRes struct {
	CommentId string `json:"comment_id"`
}

// 回复评论
type CommentReplyReq struct {
	EventId   string `json:"event_id"`
	CommentId string `json:"comment_id"`
	Oid       string `json:"oid"`
	Comment   string `json:"comment"`
	// --story=867397871
	ActivityID string `json:"activity_id"`
	ToOid      string `json:"to_oid"` // only set this field when want get to_nick
}

type CommentReplyRes struct {
	ReplyId string `json:"reply_id"`
}

type CommentTopReq struct {
	EventId   string `json:"event_id"`
	CommentId string `json:"comment_id"`
	Oid       string `json:"oid"`
	Top       bool   `json:"top"`
	// --story=867397871
	ActivityID string `json:"activity_id"`
}

type CommentLikeReq struct {
	EventId   string `json:"event_id"`
	CommentId string `json:"comment_id"`
	ReplyId   string `json:"reply_id"`
	Oid       string `json:"oid"`
	Like      bool   `json:"like"`
	Type      int    `json:"type"`
	// --story=867397871
	ActivityID string `json:"activity_id"`
}

type CommentDeleteReq struct {
	EventId   string `json:"event_id"`
	CommentId string `json:"comment_id"`
	Oid       string `json:"oid"`
	// --story=867397871
	ActivityID string `json:"activity_id"`
	Type       int    `json:"type"` // 1 comment, 2 reply
	ReplyId    string `json:"reply_id"`
}

// 首页个人数据
type FirstPageUserDataReq struct {
	Oid string `json:"oid"`
}

type FirstPageUserDataRes struct {
	DonateTotalDays  int   `json:"donate_total_days"`
	DonateTotalSteps int64 `json:"donate_total_steps"`
	DonateTotalMoney int64 `json:"donate_total_money"`
}

// 首页队伍数据
type FirstPageEventsReq struct {
	Oid     string `json:"oid"`
	UninId  string `json:"unin_id"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	EidType int    `json:"eid_type"`
}

type FirstPageUser struct {
	Nick     string  `json:"nick"`
	Head     string  `json:"head"`
	Rank     int     `json:"rank"`
	Steps    int64   `json:"steps"`
	Distance float32 `json:"distance"`
	Donate   int64   `json:"donate"`
}

type FirstPageEventInfo struct {
	Eid            string  `json:"eid"`
	Rid            int     `json:"rid"`
	EidType        int     `json:"eid_type"`
	EventName      string  `json:"event_name"`
	RouteName      string  `json:"route_name"`
	RoutePointId   int     `json:"route_point_id"`
	RoutePointHead string  `json:"route_point_head"`
	Distance       float32 `json:"distance"`
	Days           int     `json:"days"`
	RoutePoint     int     `json:"route_point"`
	CompleteRatio  float32 `json:"complete_ratio"`
}

type EventList struct {
	User      FirstPageUser      `json:"user"`
	EventInfo FirstPageEventInfo `json:"event_info"`
}

type FirstPageEventRes struct {
	Page       int         `json:"page"`
	Size       int         `json:"size"`
	ListSize   int         `json:"list_size"`
	CreateSize int         `json:"create_size"`
	JoinSize   int         `json:"join_size"`
	List       []EventList `json:"list"`
}

// 周报小组列表
type GetWeeklyTabListReq struct {
	Oid     string `json:"oid"`
	EventID string `json:"event_id"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type Tab struct {
	EventID   string `json:"event_id"`
	EventName string `json:"event_name"`
}

type GetWeeklyTabListRes struct {
	Count int   `json:"count"`
	List  []Tab `json:"list"`
}

// 获取小组周报
type GetWeeklyDataReq struct {
	Oid     string `json:"oid"`
	EventID string `json:"event_id"`
}

type WeeklyRoute struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`     // 关卡名称
	Distance float32 `json:"distance"` // 单位关卡距离，公里
	Status   int     `json:"status"`   // 当前关卡是否完成，0 表示没完成，1 表示完成
}

type OtherStepDiffer struct {
	Head       string `json:"head"`
	StepDiffer int64  `json:"step_differ"` // 步数差
}

type MyWeekly struct {
	Nick       string           `json:"nick"`
	Head       string           `json:"head"`
	Rank       int              `json:"rank"`
	TotalSteps int64            `json:"total_steps"`
	BeforeRank *OtherStepDiffer `json:"before_rank"` // 前一名数据(可为空)
	AfterRank  *OtherStepDiffer `json:"after_rank"`  // 后一名数据(可为空)
}

type StepsLine struct {
	Head     string  `json:"head"`
	Point    []int64 `json:"point"`     // 每天的步数,7个点
	IsMyself bool    `json:"is_myself"` // 是否我自己
}

type GetWeeklyDataRes struct {
	StartDate      string        `json:"start_date"`
	EndDate        string        `json:"end_date"`
	EventName      string        `json:"event_name"`
	TotalDistance  float64       `json:"total_distance"` // 行进路程
	RouteID        int           `json:"route_id"`
	RouteList      []WeeklyRoute `json:"route_list"`
	MyWeekly       MyWeekly      `json:"my_weekly"`
	StepsLineChart []StepsLine   `json:"steps_line_chart"` // 折线图
}

// 获取小队统计(每周步数排名)
type GetWeeklyStepsRankReq struct {
	EventID string `json:"event_id"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type WeeklySteps struct {
	Nick            string `json:"nick"`
	Head            string `json:"head"`
	DayAverageSteps int64  `json:"day_average_steps"` // 日均步数
	StepsOverTenK   int    `json:"steps_over_ten_k"`  // 一周内捐出的步数>10000的天数
}

type GetWeeklyStepsRankRes struct {
	Count int           `json:"count"`
	List  []WeeklySteps `json:"list"`
}

// 回补步数
type CoverStepsReq struct {
	Oid          string `json:"oid"`
	UniId        string `json:"uni_id"`
	Appid        string `json:"appid"`
	EcryptedData string `json:"ecrypted_data"`
	EcryptedIv   string `json:"ecrypted_iv"`
}

type CoverStepsRes struct {
	//CoverSteps       int64             `json:"cover_steps"`
	//CoverDay         int               `json:"cover_day"`
	//Donated          int64             `json:"donated"`
	//DonatedSteps     int64             `json:"donated_steps"`
	//ArriveViews      []DonateViewPoint `json:"arrive_views"`
	//CheckPointSteped int64             `json:"check_point_steped"`
	NeedCover bool `json:"need_cover"`
}

// 获取资金记录
type GetCapitalRecordReq struct {
	EventID   string `json:"event_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Page      int    `json:"page"`
	Size      int    `json:"size"`
}

type CapitalRecord struct {
	Oid        string `json:"oid"`
	Head       string `json:"head"`
	Nick       string `json:"nick"`
	Money      int64  `json:"money"`
	Date       string `json:"date"`
	RecordType int    `json:"record_type"` // 0、充值 1、配捐
	IsRecharge bool   `json:"is_recharge"`
}

type GetCapitalRecordRes struct {
	Count            int             `json:"count"`
	StartTime        string          `json:"start_time"`        // 活动创建时间
	TotalDonate      int64           `json:"total_donate"`      // 公益包总金额
	SurplusDonate    int64           `json:"surplus_donate"`    // 公益包剩余金额
	TotalRedPacket   int64           `json:"total_redpacket"`   // 红包总金额
	SurplusRedPacket int64           `json:"surplus_redpacket"` // 红包剩余金额
	List             []CapitalRecord `json:"list"`
}

type ProfitShareCallBackReq struct {
	WXPFData string `json:"wx_pf_data"`
}

type GetProjecetReq struct {
	Pid int `json:"pid"`
}

type GetProjectRes struct {
	ProjectInfo ProjInfoStruct `json:"project_info"`
}

type RedEnvelopeReq struct {
	Eid         string `json:"eid"`
	Oid         string `json:"oid"`
	ReceiverOid string `json:"receiver_oid"`
	Code        string `json:"code"`
}

type RedEnvelopeRes struct {
	Rid        string `json:"rid"`
	Nick       string `json:"nick"`
	Head       string `json:"head"`
	SenderNick string `json:"sender_nick"`
	SenderHead string `json:"sender_head"`
	Status     int    `json:"status"`
	Money      int    `json:"money"`
	Leaf       int    `json:"leaf"`
	ProjId     int    `json:"proj_id"`
	ProjImg    string `json:"proj_img"`
	ProjName   string `json:"proj_name"`
	ProjTarg   string `json:"proj_targ"`
	OrgId      string `json:"org_id"`
}

// 取消领取的红包
type CancelEnvelopeReq struct {
	Rid string `json:"rid"`
}

type CancelEnvelopeRes struct {
	Status int `json:"status"`
}

// 转账操作
type TransferReq struct {
	// 加密后的rid
	Rid    string `json:"rid"`
	ToUser bool   `json:"to_user"`
}

type TransferRes struct {
	Status int `json:"status"`
}

type GetHavePacketRecordReq struct {
	Eid string `json:"eid"`
	Oid string `json:"oid"`
}

type PacketRecord struct {
	CheckPoint int `json:"check_point"`
	ViewPoint  int `json:"view_point"`
	PacketType int `json:"packet_type"` // 景点包类型 1 表示公益包，2 表示红包
}

type GetHavePacketRecordRes struct {
	PacketRecords []PacketRecord `json:"packet_records"`
}

// 柱状图数据
type GetBarGraphReq struct {
	Oid        string `json:"oid"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	IsHomePage bool   `json:"is_home_page"`
}

type UserSteps struct {
	Date  string `json:"date"`
	Steps int64  `json:"steps"`
}

type LightCity struct {
	Num   int      `json:"num"`
	Names []string `json:"names"`
}

type GetBarGraphRes struct {
	Nick       string      `json:"nick"`
	Head       string      `json:"head"`
	Days       int64       `json:"days"`
	RankSteps  int64       `json:"rank_steps"`
	AverageDay int64       `json:"average_day"`
	Rank       int64       `json:"rank"`
	TotalRank  int64       `json:"total_rank"`
	LightCity  LightCity   `json:"light_city"`
	Steps      []UserSteps `json:"steps"`
	IsClickIn  bool        `json:"is_click_in"`
}

// 首页我的小队(新版)
type GetHomePageEventsReq struct {
	Oid     string `json:"oid"`
	UninId  string `json:"unin_id"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	EidType int    `json:"eid_type"`
}

type HomePageBefore struct {
	Oid          string  `json:"oid"`
	Nick         string  `json:"nick"`
	Head         string  `json:"head"`
	Walked       float64 `json:"walked"`
	IsTeamLeader bool    `json:"is_team_leader"`
}

type HomePagePacket struct {
	Type     string  `json:"type"`     // 景点包类型 1 表示公益包，2 表示红包
	Distance float32 `json:"distance"` // 景点距离起点的距离，单位米
	OrderId  int     `json:"vid"`      // 第几个景点
	LatLng   string  `json:"latLng"`   // 景点经纬度
	Position string  `json:"position"`
}

type HomePageUser struct {
	Nick         string          `json:"nick"`
	Head         string          `json:"head"`
	Rank         int             `json:"rank"`
	IsCreator    bool            `json:"is_creator"`
	IsTeamLeader bool            `json:"is_team_leader"`
	IsFinished   bool            `json:"is_finished"`
	Walked       float64         `json:"walked"`
	Before       *HomePageBefore `json:"before"`
	Packet       *HomePagePacket `json:"packet"`
}

type HomePageEventInfo struct {
	EidType    int    `json:"eid_type"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	TotalUsers int    `json:"total_users"`
	RouteID    int    `json:"route_id"`
	LevelID    int    `json:"level_id"`
	LevelName  string `json:"level_name"`
	Level      int    `json:"level"`
	Status     int    `json:"status"`
}

type WeekRank struct {
	Nick   string `json:"nick"`
	Head   string `json:"head"`
	Step   int64  `json:"step"`
	Rank   int    `json:"rank"`
	IsSelf bool   `json:"is_self"`
}

type HomePageList struct {
	User      HomePageUser      `json:"user"`
	EventInfo HomePageEventInfo `json:"event_info"`
	WeekRank  []WeekRank        `json:"week_rank"`
}

type GetHomePageEventsRes struct {
	Total       int            `json:"total"`
	CreateSize  int            `json:"create_size"`
	JoinSize    int            `json:"join_size"`
	IsGlobal    bool           `json:"is_global"`
	UserRouteID string         `json:"user_routeid"` // 最近参与的大地图活动
	RouteID     string         `json:"routeid"`      // 当前大地图的活动ID
	List        []HomePageList `json:"list"`
}

// 首页我的小队(新版)
type GetHomePageEventsV2Req struct {
	Oid     string `json:"oid"`
	UninId  string `json:"unin_id"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	EidType int    `json:"eid_type"`
}

type HomePageBeforeV2 struct {
	Oid          string  `json:"oid"`
	Nick         string  `json:"nick"`
	Head         string  `json:"head"`
	Walked       float64 `json:"walked"`
	IsTeamLeader bool    `json:"is_team_leader"`
}

type HomePagePacketV2 struct {
	Type     string  `json:"type"`     // 景点包类型 1 表示公益包，2 表示红包
	Distance float32 `json:"distance"` // 景点距离起点的距离，单位米
	OrderId  int     `json:"vid"`      // 第几个景点
	LatLng   string  `json:"latLng"`   // 景点经纬度
	Position string  `json:"position"`
}

type HPRank struct {
	Oid    string  `json:"oid"`
	Nick   string  `json:"nick"`
	Head   string  `json:"head"`
	Walked float64 `json:"walked"`
	Rank   int     `json:"rank"`
	IsSelf bool    `json:"is_self"`
}

type HomePageUserV2 struct {
	Nick         string            `json:"nick"`
	Head         string            `json:"head"`
	Rank         int               `json:"rank"`
	IsCreator    bool              `json:"is_creator"`
	IsTeamLeader bool              `json:"is_team_leader"`
	IsFinished   bool              `json:"is_finished"`
	Walked       float64           `json:"walked"`
	Before       *HomePageBeforeV2 `json:"before"`
	Packet       *HomePagePacketV2 `json:"packet"`
	List         []HPRank          `json:"list"`
}

type HomePageEventInfoV2 struct {
	EidType    int    `json:"eid_type"`
	VerType    int    `json:"ver_type"`
	WeekID     string `json:"week_id"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	TotalUsers int    `json:"total_users"`
	RouteID    int    `json:"route_id"`
	LevelID    int    `json:"level_id"`
	LevelName  string `json:"level_name"`
	Level      int    `json:"level"`
	Status     int    `json:"status"`
}

type WeekRankV2 struct {
	Nick   string `json:"nick"`
	Head   string `json:"head"`
	Step   int64  `json:"step"`
	Rank   int    `json:"rank"`
	IsSelf bool   `json:"is_self"`
}

type HomePageListV2 struct {
	User      HomePageUserV2      `json:"user"`
	EventInfo HomePageEventInfoV2 `json:"event_info"`
	WeekRank  []WeekRankV2        `json:"week_rank"`
}

type GetHomePageEventsV2Res struct {
	Total       int              `json:"total"`
	CreateSize  int              `json:"create_size"`
	JoinSize    int              `json:"join_size"`
	IsGlobal    bool             `json:"is_global"`
	UserRouteID string           `json:"user_routeid"` // 最近参与的大地图活动
	RouteID     string           `json:"routeid"`      // 当前大地图的活动ID
	List        []HomePageListV2 `json:"list"`
}

// 首页我的小队(新版)
type GetHomePageEventsV3Req struct {
	Oid    string `json:"oid"`
	UninId string `json:"unin_id"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}

type HPRedEnvelope struct {
	Nick string `json:"nick"`
	Head string `json:"head"`
}

type HomePageEventInfoV3 struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	TotalUsers int              `json:"total_users"`
	WeekID     string           `json:"week_id"`
	Rank       int              `json:"rank"`
	TeamType   int              `json:"team_type"`
	Top        []GetHomeUserRes `json:"top"`
	Last       []GetHomeUserRes `json:"last"`
	Red        []HPRedEnvelope  `json:"red"`
	TopSize    int64            `json:"top_size"`
	LastSize   int64            `json:"last_size"`
	RedSize    int64            `json:"red_size"`
	RouteId    int              `json:"route_id"`
}

type GetHomePageEventsV3Res struct {
	Total int                   `json:"total"`
	Leaf  int64                 `json:"leaf"`
	List  []HomePageEventInfoV3 `json:"list"`
}

// 获取公益包报告（周报/月报）
type GetReportGongyiReq struct {
	Oid       string `json:"oid"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Page      int    `json:"page"`
	Size      int    `json:"size"`
}

type ReportGongyi struct {
	Targ string `json:"targ"`
	Num  int    `json:"num"`
	Qunt string `json:"qunt"`
	Nm   string `json:"nm"`
}

type GetReportGongyiRes struct {
	Count int            `json:"count"`
	Total int            `json:"total"`
	List  []ReportGongyi `json:"list"`
}

// 获取红包报告（周报/月报）
type GetReportHongbaoReq struct {
	Oid       string `json:"oid"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Page      int    `json:"page"`
	Size      int    `json:"size"`
}

type ReportHongbao struct {
	Num   int    `json:"num"`
	Money int    `json:"money"`
	Nick  string `json:"nick"`
	Head  string `json:"head"`
}

type GetReportHongbaoRes struct {
	Count int             `json:"count"`
	Total int             `json:"total"`
	List  []ReportHongbao `json:"list"`
}

// 获取小队报告（周报/月报）
type GetReportEventReq struct {
	Oid       string `json:"oid"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Page      int    `json:"page"`
	Size      int    `json:"size"`
}

type ReportEvent struct {
	Rank   int    `json:"rank"`
	Member int    `json:"member"`
	Name   string `json:"name"`
}

type GetReportEventRes struct {
	Count int           `json:"count"`
	List  []ReportEvent `json:"list"`
}

// 获取pk信息
type GetPkReq struct {
	UserId string `json:"oid"`
	//EventId   string `json:"event_id"`
	PkId      string `json:"pk_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GetPkRes struct {
	PkId        string      `json:"pk_id"`
	UserId      string      `json:"user_id"`
	UserNick    string      `json:"nick"`
	UserHead    string      `json:"head"`
	PkUserId    string      `json:"pk_oid"`
	PkUserNick  string      `json:"pk_nick"`
	PkUserHead  string      `json:"pk_head"`
	StartPkDate string      `json:"start_pk_date"`
	EndPkDate   string      `json:"end_pk_date"`
	EventId     string      `json:"event_id"`
	List        []UserSteps `json:"list"`
	PkList      []UserSteps `json:"pk_list"`
}

type EditPkReq struct {
	UserId   string `json:"oid"`
	PkUserId string `json:"pk_oid"`
	EventId  string `json:"event_id"`
	PkDays   int    `json:"pk_days"`
}

type EditPkRes struct {
	Status     int    `json:"status"`
	PkUserId   string `json:"pk_oid"`
	PkUserNick string `json:"pk_nick"`
	PkUserHead string `json:"pk_head"`
}

type GetPkListReq struct {
	EventId string `json:"event_id"`
	UserId  string `json:"oid"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type GetPkListRes struct {
	Page      int          `json:"page"`
	Size      int          `json:"size"`
	Steps     int64        `json:"steps"`
	EventName string       `json:"event_name"`
	Data      []PkListItem `json:"data"`
}

type PkListItem struct {
	UserId   string `json:"oid"`
	Nick     string `json:"nick"`
	Head     string `json:"head"`
	Steps    int64  `json:"steps"`
	AvgSteps int64  `json:"avg_steps"`
	IsPk     bool   `json:"is_pk"` //是否已经是pk对象了
}

type GetPkUserReq struct {
	UserId string `json:"oid"`
}

type GetPkUserRes struct {
	Eid   string `json:"eid"`
	PkOid string `json:"pk_oid"`
}

type GetPkShareIdReq struct {
	UserId   string `json:"oid"`
	PkUserId string `json:"pk_user_id"`
	EventId  string `json:"event_id"`
	SType    int    `json:"s_type"`
}

type GetPkShareIdRes struct {
	ShareId string `json:"share_id"`
	Date    string `json:"date"`
}

/* 消息模板 */
type GetSubMsgStatus struct {
	MsgTempleId string `json:"msg_id"`
	Status      int    `json:"status"`
	CreateTime  string `json:"create_time"`
	UpdateTime  string `json:"update_time"`
}

type GetSubMsgStatusReq struct {
	UserId       string   `json:"oid"`
	MsgTempleIds []string `json:"msg_ids"` //空数组则传所有消息模板
}

type GetSubMsgStatusRes struct {
	UserId    string            `json:"oid"`
	MsgStatus []GetSubMsgStatus `json:"msg_status"`
}

// 大地图
type GetBigMapReq struct {
	UserId  string `json:"oid"`
	Pid     string `json:"pid"`
	RouteId int    `json:"route_id"`
}

type NearbyUser struct {
	UserId string `json:"oid"`
	Steps  int64  `json:"steps"`
	Nick   string `json:"nick"`
	Head   string `json:"head"`
}

type BigMapInfoRes struct {
	Pid         string       `json:"pid"`
	UserId      string       `json:"oid"`
	Steps       int64        `json:"steps"`
	JoinDate    string       `json:"join_date"`
	NearbyUsers []NearbyUser `json:"nearby"`
}

type JoinBigMapReq struct {
	Oid string `json:"oid"`
}

type JoinBigMapRes struct {
	Exist bool   `json:"exist"`
	Pid   string `json:"pid"`
}

type GetJoinMapHeadReq struct {
	RouteId int `json:"route_id"`
}

type GetJoinMapHeadRes struct {
	Count int64    `json:"count"`
	Heads []string `json:"heads"`
}

type BigMapDonateStepsReq struct {
	Oid          string `json:"oid"`
	Appid        string `json:"appid"`
	EcryptedData string `json:"ecrypted_data"`
	EcryptedIv   string `json:"ecrypted_iv"`
	UniId        string `json:"uni_id"`
	Authorize    bool   `json:"authorize"`
}

type BigMapDonateStepsRes struct {
	Nick         string  `json:"nick"`
	Head         string  `json:"head"`
	Total        int64   `json:"total"`
	Rank         int64   `json:"rank"`
	WeekSteps    int64   `json:"week_steps"`
	WeekDistance float32 `json:"week_distance"`
	DonateStep   int64   `json:"donate_step"`
	TodayStep    int64   `json:"today_step"`
	WeekId       string  `json:"week_id"`
	IsBlack      bool    `json:"is_black"`

	//
	Pid string `json:"pid"`
}

// 同期数据
type GetSamePeriodDataReq struct {
	Oid     string `json:"oid"`
	RouteId int    `json:"route_id"`
	Pid     string `json:"pid"`
}

type FinishCert struct {
	Name string `json:"name"`
	Time string `json:"time"`
}

type SPRoute struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Steps  int64  `json:"steps"`
}

type GetSamePeriodDataRes struct {
	Nick       string       `json:"nick"`
	Head       string       `json:"head"`
	Day        int          `json:"day"`
	Walker     float64      `json:"walker"`
	Rank       int          `json:"rank"`
	TodayStep  int64        `json:"today_step"`
	Steps      int64        `json:"steps"`
	Cert       bool         `json:"cert"`
	FinishCert []FinishCert `json:"finish_cert"`
	Route      []SPRoute    `json:"route"`
}

// 弹幕数据
type GetBarrageDataReq struct {
	EventID string `json:"event_id"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type BarrageData struct {
	Head    string `json:"head"`
	Content string `json:"content"`
}

type GetBarrageDataRes struct {
	Count int           `json:"count"`
	List  []BarrageData `json:"list"`
}

type GetUserJoinStatusReq struct {
	Oid string `json:"oid"`
}

type GetUserJoinStatusRes struct {
	Join bool `json:"join"` // true 表示加入
}

type GetPkShareStatusReq struct {
	ShareId string `json:"share_id"`
	Oid     string `json:"oid"`
}

type GetPkShareStatusRes struct {
	ShareId string `json:"share_id"`
	PkOid   string `json:"pk_oid"`
	IsSelf  bool   `json:"is_self"`
	Status  int    `json:"status"` // 0 表示合法，1 表示过期，2 表示当天扫描超过次数
	Nick    string `json:"nick"`   // 分享人的nick
	Head    string `json:"head"`   // 下同
}

type GetPkHistoryReq struct {
	Oid  string `json:"oid"`
	Page int    `json:"page"`
	Size int    `json:"size"`
}

type PKHistoryInfo struct {
	Nick      string `json:"nick"`
	Head      string `json:"head"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Score     int    `json:"score"`
	PkScore   int    `json:"pk_score"`
}

type GetPKHistoryRes struct {
	Nick string          `json:"nick"`
	Head string          `json:"head"`
	List []PKHistoryInfo `json:"list"`
}

// 队友步数排行
type GetTeammateRankReq struct {
	Oid     string `json:"oid"`
	RouteID string `json:"route_id"`
}

type TeammateRank struct {
	Nick       string `json:"nick"`
	Head       string `json:"head"`
	Steps      int64  `json:"steps"`
	City       int    `json:"city"`
	TodaySteps int64  `json:"today_steps"`
	IsSelf     bool   `json:"is_self"`
}

type GetTeammateRankRes struct {
	Rank          []TeammateRank `json:"rank"`
	OtherTeammate int            `json:"other_teammate"`
}

// 全部队友步数排行
type GetAllTeammateRankReq struct {
	Oid     string `json:"oid"`
	RouteID string `json:"route_id"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type GetAllTeammateRankRes struct {
	Rank []TeammateRank `json:"rank"`
}

type GetBigMapHeadReq struct {
	Oid string `json:"oid"`
	Pid string `json:"pid"`
}

type BigMapNickHeadData struct {
	Nick   string `json:"nick"`
	Head   string `json:"head"`
	IsSelf bool   `json:"is_self"`
	Step   int64  `json:"step"`
}

type GetBigMapHeadRes struct {
	K10  []BigMapNickHeadData `json:"10"`
	K40  []BigMapNickHeadData `json:"40"`
	K80  []BigMapNickHeadData `json:"80"`
	K100 []BigMapNickHeadData `json:"100+"`
}

type GetBigMapHeadReqV2 struct {
	Oid string `json:"oid"`
	Pid string `json:"pid"`
}

type BigMapNickHeadDataV2 struct {
	Distance int    `json:"distance"`
	Nick     string `json:"nick"`
	Head     string `json:"head"`
	Step     int64  `json:"step"`
	IsSelf   bool   `json:"is_self"`
	UserNum  int    `json:"user_num"`
}

type GetBigMapHeadResV2 struct {
	D100 []BigMapNickHeadDataV2 `json:"100-"`
	I100 []BigMapNickHeadDataV2 `json:"100+"`
}

type GetGlobalRank struct {
	Openid  string `json:"oid"`
	RouteId string `json:"routeid"`
}

type GetGlobalRankList struct {
	RouteId  string `json:"routeid"`
	RankPage int64  `json:"rankpage"`
	RankSize int64  `json:"ranksize"`
}

type ResGlobalRank struct {
	Oid        string `json:"oid,omitempty"`
	Rank       int64  `json:"rank"`
	WeekSteps  int64  `json:"weekStep"`
	TodaySteps int64  `json:"todayStep"`
	Head       string `json:"head"`
	Nick       string `json:"nick"`
}

type ResGlobalRankList struct {
	Page  int64           `json:"page"`
	Size  int64           `json:"size"`
	Total int64           `json:"total"`
	List  []ResGlobalRank `json:"list"`
}

// 大一块走活动页面
type GlobalWeekDataReq struct {
	Oid         string `json:"oid"`
	IsMsgPush   bool   `json:"is_msg_push"`
	ArriveNum   int    `json:"arrive_num"`
	MsgWeekID   string `json:"msg_week_id"`
	IsMatchPush bool   `json:"is_match_push"`
	MatchDate   string `json:"match_date"`
}

type GWDMapData struct {
	WeekID   string `json:"week_id"`
	RouteID  int    `json:"route_id"`
	Week     int    `json:"week"`
	Name     string `json:"name"`
	Distance int    `json:"distance"`
	Day      int    `json:"day"`
	UserNum  int64  `json:"user_num"`
}

type GWDLead struct {
	Oid      string  `json:"oid"`
	Head     string  `json:"head"`
	Walked   float64 `json:"walked"`
	Step     int64   `json:"step"`
	Rank     int64   `json:"rank"`
	IsMyself bool    `json:"is_myself"`
	IsBlack  bool    `json:"is_black"`
}

type GWDPoint struct {
	Interval int   `json:"interval"`
	Num      int64 `json:"num"`
}

type GWDChart struct {
	Leads  []GWDLead  `json:"leads"`
	Points []GWDPoint `json:"points"`
	Animal []int64    `json:"animal"`
}

type GWDMyData struct {
	Nick       string `json:"nick"`
	Head       string `json:"head"`
	Percentage int64  `json:"percentage"`
	Step       int64  `json:"step"`
	Rank       int64  `json:"rank"`
	TodaySteps int64  `json:"today_steps"`
	UpdateTime string `json:"update_time"`
	IsBlack    bool   `json:"is_black"`
}

type MatchDonate struct {
	CompanyId    string `json:"company_id"`
	CompanyName  string `json:"company_name"`
	CompanyLogo  string `json:"company_logo"`
	MatchEventId string `json:"match_event_id"`
	MatchStatus  int    `json:"match_status"`
	TotalSteps   int64  `json:"total_steps"`
	TotalDays    int    `json:"total_days"`
	TotalTimes   int    `json:"total_times"`
	TotalMoney   int64  `json:"total_money"`
}

type GlobalWeekDataRes struct {
	IsDonate bool        `json:"is_donate"`
	MapData  GWDMapData  `json:"map_data"`
	Chart    GWDChart    `json:"chart"`
	MyData   *GWDMyData  `json:"my_data"`
	Match    MatchDonate `json:"match_donate"`
}

type CreateNewEventReq struct {
	EventName  string `json:"event_name"`
	CreaterOrg string `json:"creater_orgn"`
	EventDesc  string `json:"event_desc"`
	CreaterID  string `json:"creater_id"`
	EventType  string `json:"event_type"`
	AppId      string `json:"appid"`
}

type CreateNewEventRes struct {
	EventId string `json:"event_id"`
}

type GetWeekTeamInfoReq struct {
	Eid    string `json:"eid"`
	Oid    string `json:"oid"`
	Offset int32  `json:"offset"`
	Size   int32  `json:"size"`
}

type WeekEventInfo struct {
	Name        string `json:"name"`
	UserNum     int    `json:"user_num"`
	StartDate   string `json:"start_date"`
	WeekNum     int    `json:"week_num"`
	IsRecharge  bool   `json:"is_recharge"`
	MapId       int    `json:"map_id"`
	IsCreator   bool   `json:"is_creator"`
	CreatorNick string `json:"creator_nick"`
	WeekId      string `json:"week_id"`
	EventStat   int    `json:"event_stat"`
	EventType   int    `json:"event_type"`
	IsWatch     bool   `json:"is_watch"`
}

type WeekMyInfo struct {
	IsJoin   bool    `json:"is_join"`
	Percent  float32 `json:"percent"`
	IsWhite  bool    `json:"is_white"`
	IsDonate bool    `json:"is_donate"`
	Nick     string  `json:"nick"`
	Head     string  `json:"head"`
	// new support field
	Rank       int     `json:"rank"`
	TodaySteps int64   `json:"today_steps"`
	WeekSteps  []int64 `json:"week_steps"`
}

type ListData struct {
	Rank         int     `json:"rank"`
	Nick         string  `json:"nick"`
	Head         string  `json:"head"`
	Leaf         int64   `json:"leaf"`
	TodaySteps   int64   `json:"today_steps"`
	WeekSteps    []int64 `json:"week_steps"`
	IsSelf       bool    `json:"is_self"`
	IsTeamLeader bool    `json:"is_team_leader"`
	Show         bool    `json:"show"`
	Oid          string  `json:"Oid"`
}

type WeekRankInfo struct {
	Days       int        `json:"days"`
	TotalUsers int        `json:"total_users"`
	List       []ListData `json:"list"`
}

type AnimalInfo struct {
	Eid        string      `json:"eid"`
	TeamType   int         `json:"team_type"`
	AnimalList []AnimalNum `json:"animal_list"`
	UpdateTime string      `json:"update_time"`
}

type GetWeekTeamInfoRes struct {
	EventInfo WeekEventInfo `json:"event_info"`
	MyInfo    WeekMyInfo    `json:"my_info"`
	Animal    AnimalInfo    `json:"animal_info"`
	Rank      WeekRankInfo  `json:"rank"`
}

// 获取证书
type GetCertificateReq struct {
	WeekID string `json:"week_id"`
	Oid    string `json:"oid"`
}

type GetCertificateRes struct {
	IsBattle  bool    `json:"is_battle"`
	CertTitle string  `json:"cert_title"`
	RouteName string  `json:"route_name"`
	Week      int     `json:"week"`
	RouteID   int     `json:"route_id"`
	MapName   string  `json:"map_name"`
	Nick      string  `json:"nick"`
	Head      string  `json:"head"`
	Round     int     `json:"round"`
	Walked    float64 `json:"walked"`
	Exceed    int     `json:"exceed"`
	Date      string  `json:"date"`
}

// 获取证书v2
type GetCertificateV2Req struct {
	WeekID    string `json:"week_id"`
	Oid       string `json:"oid"`
	IsMsgPush bool   `json:"is_msg_push"`
}

type GetCertificateV2Res struct {
	Week             int    `json:"week"`
	MapTitle         string `json:"map_title"`
	MapName          string `json:"map_name"`
	MapDistance      int    `json:"map_distance"`
	MapDistanceMeter int    `json:"map_distance_meter"`
	Nick             string `json:"nick"`
	Head             string `json:"head"`
	Exceed           int64  `json:"exceed"`
	FinishTime       string `json:"finish_time"`
	Date             string `json:"date"`
}

type GetWeekNearbyReq struct {
	Oid     string `json:"oid"`
	RouteId string `json:"route_id"`
	HeadNum int64  `json:"head_num"` // 头部个数
	TailNum int64  `json:"tail_num"` // 尾部个数
}

type GetWeekNearbyRes struct {
	MyRank   int64           `json:"my_rank"`
	TotalNum int64           `json:"total_num"`
	List     []ResGlobalRank `json:"list"`
}

type GetHomeUserRes struct {
	Head string `json:"head"`
	Step int64  `json:"step"`
}

// 是否可以充值
type CheckRechargeReq struct {
	Eid string `json:"eid"`
	Oid string `json:"oid"`
}

type CheckRechargeRes struct {
	IsRecharge bool `json:"is_recharge"`
}

// 获取红包相关数据
type GetRedEnvelopeDataReq struct {
	Eid string `json:"eid"`
}

type RechargeInfo struct {
	Code string `json:"code"`
	Oid  string `json:"oid"`
	Nick string `json:"nick"`
	Head string `json:"head"`
}

type GetRedEnvelopeDataRes struct {
	RechargeList []RechargeInfo `json:"recharge_list"`
}

// 获取用户
type GetUserListReq struct {
	EventId string `json:"eid"`
	Oid     string `json:"oid"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type UserInfo struct {
	Oid    string `json:"oid"`
	Nick   string `json:"nick"`
	Head   string `json:"head"`
	IsSelf bool   `json:"is_self"`
}

func (m *UserInfo) String() string {
	return fmt.Sprintf("{oid: %v, nick: %v, head: %v, self: %v}", m.Oid, m.Nick, m.Head, m.IsSelf)
}

type GetUserListRes struct {
	Page int        `json:"page"`
	Size int        `json:"size"`
	List []UserInfo `json:"list"`
}

// 移出用户
type RemoveUserReq struct {
	Eid         string `json:"eid"`
	Oid         string `json:"oid"`
	OperatorOid string `json:"operator_oid"`
	Week        string `json:"week"`
}

type RemoveUserRes struct {
	Status int `json:"status"`
}

// 柱状图数据v3
type GetBarGraphV3Req struct {
	Oid       string `json:"oid"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	IsWeekly  bool   `json:"is_weekly"`
}

type BGV3MapData struct {
	IsJoin   bool `json:"is_join"`
	Week     int  `json:"week"`
	IsFinish bool `json:"is_finish"`
	MapID    int  `json:"map_id"`
}

type GetBarGraphV3Res struct {
	Nick       string      `json:"nick"`
	Head       string      `json:"head"`
	Days       int64       `json:"days"`
	AverageDay int64       `json:"average_day"`
	TotalStep  int64       `json:"total_step"`
	Steps      []UserSteps `json:"steps"`
	MapData    BGV3MapData `json:"map_data"`
}

// 获取小队运动报告（动物版本）
type GetTeamReportReq struct {
	Week      string    `json:"week"`
	Oid       string    `json:"oid"`
	EventList []TREvent `json:"event_list"`
	IsMsgPush bool      `json:"is_msg_push"`
}

type TREvent struct {
	Eid       string `json:"eid"`
	AnimalTop int    `json:"animal_top"`
}

type AnimalNum struct {
	AnimalType int `json:"animal_type"`
	Num        int `json:"num"`
}

type AnimalTop struct {
	Nick string `json:"nick"`
	Head string `json:"head"`
	Num  int    `json:"num"`
}

type TeamReport struct {
	Eid        string      `json:"eid"`
	TeamName   string      `json:"team_name"`
	TeamType   int         `json:"team_type"`
	AnimalTop  int         `json:"animal_top"`
	AnimalList []AnimalNum `json:"animal_list"`
	TopList    []AnimalTop `json:"top_list"`
}

type GetTeamReportRes struct {
	List []TeamReport `json:"list"`
}

type CreatePkReq struct {
	AppId         string `json:"app_id"`
	CreateOid     string `json:"create_oid"`
	JoinOid       string `json:"join_oid"`
	UnionOid      string `json:"union_oid"`
	EncryptedData string `json:"encrypted_data"`
	Iv            string `json:"iv"`
}

type CreatePkRes struct {
	Eid        string `json:"eid"`
	CreateTime string `json:"create_time"`
	InPk       bool   `json:"in_pk"`
	PkType     int    `json:"pk_type"`
}

// 趣味PK: 加入、退出PK
type GetPKJoinReq struct {
	Eid  string `json:"eid"`
	Oid  string `json:"oid"`
	Join bool   `json:"join"`
}

// 获取pk详情
type GetPKDetailsReq struct {
	WeekID string `json:"week_id"`
	Eid    string `json:"eid"`
	Oid    string `json:"oid"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}

type PKDetails struct {
	Oid     string      `json:"oid"`
	Nick    string      `json:"nick"`
	Head    string      `json:"head"`
	Rank    int         `json:"rank"`
	IsSelf  bool        `json:"is_self"`
	MaxStep int64       `json:"max_step"`
	Steps   []UserSteps `json:"steps"`
	Snails  int         `json:"snails"`
}

type PKMyData struct {
	Nick    string      `json:"nick"`
	Head    string      `json:"head"`
	Rank    int64       `json:"rank"`
	IsJoin  bool        `json:"is_join"`
	MaxStep int64       `json:"max_step"`
	Steps   []UserSteps `json:"steps"`
	Snails  int64       `json:"snails"`
}

type GetPKDetailsRes struct {
	CreateTime string      `json:"create_time"`
	WeekID     string      `json:"week_id"`
	WXGroupID  string      `json:"wx_group_id"`
	PKType     int         `json:"pk_type"`
	MyData     PKMyData    `json:"my_data"`
	List       []PKDetails `json:"list"`
}

// 获取pk卡片
type GetPKCardReq struct {
	Oid       string `json:"oid"`
	PKOid     string `json:"pk_oid"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type PKCard struct {
	Nick       string      `json:"nick"`
	Head       string      `json:"head"`
	IsSelf     bool        `json:"is_self"`
	TotalStep  int64       `json:"total_step"`
	Steps      []UserSteps `json:"steps"`
	AnimalList []AnimalNum `json:"animal_list"`
}

type GetPKCardRes struct {
	List []PKCard `json:"list"`
}

// GetUserPKListRequest ...
// params:
//  - Oid : user's H5 openID
//  - Week : specified week (2021-05-24"). default this week
//  - Page: start page, each page is 10 item. default 0
//  - Offset: start offset, begin from 0. default 0
//  - Size: how many items we need. default 10
type GetUserPKListRequest struct {
	Oid    string `json:"oid"`
	Week   string `json:"week"`
	Page   int    `json:"page"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
}

// GetUserPKListResponse ...
// params:
// 	- Total: total count of user's pk item
// 	- List : pk item detail list
type GetUserPKListResponse struct {
	Total int               `json:"total"`
	List  []*PKEventSummary `json:"list"`
}

// PKEventSummary ...
type PKEventSummary struct {
	EventID        string   `json:"event_id"`        // event id
	Type           int      `json:"type"`            // pk type, 1: 1vs1, 2:group pk
	TeamID         string   `json:"team_id"`         // team id
	TotalUsers     int      `json:"total_users"`     // team user cnt
	Rank           int      `json:"rank"`            // user rank
	Mtime          string   `json:"mtime"`           // data update time
	HeadPic        []string `json:"head_pic"`        // head pic
	Opponent       string   `json:"opponent"`        // if 1vs1, opponent name
	Exited         bool     `json:"exited"`          // exited
	SelfSteps      int      `json:"self_steps"`      // self step
	SelfSnails     int      `json:"self_snails"`     // self snails
	OpponentSteps  int      `json:"opponent_steps"`  // opp  step
	OpponentSnails int      `json:"opponent_snails"` // opp snails
}

// pk捐步接口
type DonatePkStepsReq struct {
	EventId       string `json:"event_id"`
	Oid           string `json:"oid"`
	Appid         string `json:"app_id"`
	EncryptedData string `json:"encrypted_data"`
	EncryptedIv   string `json:"encrypted_iv"`
	UniId         string `json:"union_oid"`
}

type DateStep struct {
	Date string `json:"date"`
	Step int64  `json:"step"`
}

type DonatePkStepsRes struct {
	Nick  string     `json:"nick"`
	Head  string     `json:"head"`
	Rank  int64      `json:"rank"`
	Steps []DateStep `json:"steps"`
}

type GetUserTodayMatchRequest struct {
	Oid        string `json:"oid"`
	ActivityId string `json:"activity_id"`
}

type MatchActivity struct {
	Exist bool `json:"exist"`

	ProjectID   string `json:"proj_id"`
	ProjectName string `json:"proj_name"`
	ProjectImg  string `json:"proj_img"`
	ProjectExt  string `json:"proj_ext"`

	EnterpriseID   string `json:"ent_id"`
	EnterpriseLogo string `json:"ent_logo"`
	EnterpriseName string `json:"ent_name"`
	EnterpriseAbbr string `json:"ent_abbr_name"`

	MatchItem   string `json:"match_item"`
	TotalMoney  int64  `json:"total_money"`
	RemainMoney int64  `json:"remain_money"`
	Status      int    `json:"status"` // 0 not start , 1 start, 2 match finish, 3 match remain
	Mode        int    `json:"mode"`   // 0 small pool , 1 big pool
	Quota       int    `json:"donate_begin_step"`

	MatchUser int   `json:"match_user"`
	MatchFund int64 `json:"match_fund"`
}

func (m *MatchActivity) String() string {
	return fmt.Sprintf("{exist: %v, proj_id: %v, proj_name: %v, proj_img: %v, proj_ext: %v, ent_id: %v, ent_logo: %v, ent_name: %v, ent_abbr_name: %v, item: %v, total_money: %v, remain_money: %v, status: %v, mode: %v, quota: %v, user: %v, fund: %v }",
		m.Exist, m.ProjectID, m.ProjectName, m.ProjectImg, m.ProjectExt, m.EnterpriseID, m.EnterpriseLogo, m.EnterpriseName, m.EnterpriseAbbr, m.MatchItem, m.TotalMoney, m.RemainMoney, m.Status, m.Mode, m.Quota, m.MatchUser, m.MatchFund)
}

type MatchUser struct {
	NickName   string     `json:"nick"`
	Head       string     `json:"head"`
	Combo      int        `json:"days"`
	Match      bool       `json:"match_flag"`
	MatchID    string     `json:"match_id"`
	Buff       int        `json:"is_buff_donate"`
	Step       int        `json:"donate_steps"`
	Fund       int        `json:"donate_money"`
	TotalFunds int64      `json:"total_funds"`
	TotalTimes int        `json:"total_times"`
	CouponFlag bool       `json:"coupons_flag"`
	LCoupons   []*Coupons `json:"love_coupons"`
	//TotalSteps int64  `json:"total_steps"`
	//TotalDays  int    `json:"total_days"`
}

func (m *MatchUser) String() string {
	return fmt.Sprintf("{nick: %v, head: %v, combo: %v, match: %v, match_id: %v, buff: %v, step: %v, fund: %v, t_fund: %v, t_times: %v}", m.NickName, m.Head, m.Combo, m.Match, m.MatchID, m.Buff, m.Step, m.Fund, m.TotalFunds, m.TotalTimes)
}

type GetUserMatchXxhSignRequest struct {
	Aid  string `json:"aid"`
	Oid  string `json:"oid"`
	Code string `json:"code"`
}

type GetUserMatchXxhSignResponse struct {
	Sign  string `json:"sign"`
	Steps int32  `json:"steps"`
	Money int32  `json:"money"`
	Dt    string `json:"dt"`
}

type GetUserTodayMatchResponse struct {
	Activity *MatchActivity `json:"act"`
	User     *MatchUser     `json:"user"`
}

// Match business request & response
type GetUserMatchRecordRequest struct {
	Oid        string `json:"oid"`
	Offset     int    `json:"offset"`
	Size       int    `json:"size"`
	ActivityId string `json:"activity_id"`
}

type HistoryMatch struct {
	ProjectImg  string `json:"proj_img"`
	ProjectName string `json:"proj_name"`
	ProjectID   string `json:"proj_id"`
	Steps       int    `json:"donate_steps"`
	Funds       int    `json:"donate_money"`
	Date        string `json:"date"`
	// --story=866983307 second dev
	Buff      bool   `json:"is_buff_donate"`
	OrgLogo   string `json:"org_logo"`
	OrgName   string `json:"org_name"`
	MatchType int    `json:"match_type"` // 1: yqz, 2: activity
	Quota     int    `json:"step_quota"`
	// activity
	ActivityID  string     `json:"activity_id"`
	Bgpic       string     `json:"bgpic"`        // 适用于活动型，活动背景头图
	IsCreator   int        `json:"is_creator"`   // 适用于活动型移动端发起的活动，1表示用户为活动创建者
	ShowSponsor int        `json:"show_sponsor"` // 适用于活动型，是否展示发起人信息（适用于移动端发起）, 0表示不展示发起人信息, 1表示展示发起人信息, 默认值为0
	MatchID     string     `json:"match_id"`
	CouponFlag  bool       `json:"coupons_flag"`
	LCoupons    []*Coupons `json:"love_coupons"`
}

func (m *HistoryMatch) String() string {
	return fmt.Sprintf("{matchID: %v, img: %v, name: %v, p_id: %v, steps: %v, money: %v, date: %v, buff: %v, orgLogo: %v, orgName: %v, type: %v, quota: %v, ac: %v}",
		m.MatchID, m.ProjectImg, m.ProjectName, m.ProjectID, m.Steps, m.Funds, m.Date, m.Buff, m.OrgLogo, m.OrgName, m.MatchType, m.Quota, m.ActivityID)
}

type GetUserMatchRecordResponse struct {
	TotalSteps int             `json:"total_steps"`
	TotalFunds int             `json:"total_money"`
	TotalTimes int             `json:"total_times"`
	List       []*HistoryMatch `json:"list"`
	// --story=866983307 second dev
	Nick string `json:"nick"`
	Head string `json:"head"`
}

type GetUserWeekMatchRequest struct {
	Oid        string `json:"oid"`
	Start      string `json:"begin_date"`
	End        string `json:"end_date"`
	ActivityId string `json:"activity_id"`
}

type WeekMatch struct {
	Date  string `json:"date"`
	Logo  string `json:"org_logo"`
	Steps int    `json:"steps"`
	Funds int    `json:"funds"`
}

func (m *WeekMatch) String() string {
	return fmt.Sprintf("{date: %v, logo: %v, steps: %v, funds: %v}", m.Date, m.Logo, m.Steps, m.Funds)
}

type GetUserWeekMatchResponse struct {
	List []*WeekMatch `json:"list"`
}

type UserMatchRequest struct {
	Oid          string `json:"oid"`
	Appid        string `json:"appid"`
	EcryptedData string `json:"ecrypted_data"`
	EcryptedIv   string `json:"ecrypted_iv"`
	UniId        string `json:"uni_id"`
	ActivityId   string `json:"activity_id"`
}

type UserMatchResponse struct {
	AvailabelSteps int `json:"available_steps"`
	Buff           int `json:"is_buff_donate"` // Normal = 1, BUFF = 2
	Steps          int `json:"donate_steps"`
	Funds          int `json:"donate_money"`
	Status         int `json:"donate_status"` // SUCCESS = 0; FAILED_MATCH_END = 1; FAILED_STEP_NOT_ENOUGH = 2; FAILED_ALREADY_MATCH = 3;
	Combos         int `json:"combo"`
	// ActivityStatus int `json:"status"`
	MatchID    string     `json:"match_id"`
	CouponFlag bool       `json:"coupons_flag"`
	LCoupons   []*Coupons `json:"love_coupons"`
}

type Coupons struct {
	Type           int    `json:"cp_type"` // 1消费券， 2商家券
	Name           string `json:"cp_name"`
	StockType      string `json:"cp_stock_type"`      // [消费券] - NORMAL：固定面额满减券批次 CUT_TO：减至券批次;  [商家券] NORMAL：固定面额满减券批次/ DISCOUNT：折扣券批次/ EXCHANGE：换购券批次
	Threashold     int    `json:"cp_cill"`            // 批次类型为NORMAL、DISCOUNT、EXCHANGE：消费门槛（单位：分）/ 批次类型为CUT_TO：可用优惠的商品最高单价（单位：分）
	Reduction      int    `json:"cp_reduction"`       // 批次类型为NORMAL：优惠金额（单位：分） / 批次类型为DISCOUNT：折扣百分比，例如：86为八六折。示例值：86 / 批次类型为EXCHANGE：单品换购价（单位：分）/ 批次类型为CUT_TO：减至后的优惠单价（单位：分）
	Scope          string `json:"cp_scope"`           // 可用范围
	Start          string `json:"cp_available_start"` // 使用起始时间
	End            string `json:"cp_available_end"`   // 领取起始时间
	EnterpriseName string `json:"enterprise_name"`
	EnterpriseLogo string `json:"enterprise_logo"`
	Result         int    `json:"send_result"` // 0：发放爱心券到微信卡包失败（终态）;1：发放爱心券到微信卡包成功（终态）;2：发放中（非终态）;3：用户已领取过该爱心券（终态）
}

type UserMatchH5Request struct {
	Oid        string `json:"oid"`
	ActivityId string `json:"activity_id"`
	Code       string `json:"code"`
}

type UserMatchH5Response struct {
	AvailabelSteps int `json:"available_steps"`
	Buff           int `json:"is_buff_donate"` // Normal = 1, BUFF = 2
	Steps          int `json:"donate_steps"`
	Funds          int `json:"donate_money"`
	Status         int `json:"donate_status"` // SUCCESS = 0; FAILED_MATCH_END = 1; FAILED_STEP_NOT_ENOUGH = 2; FAILED_ALREADY_MATCH = 3;
	Combos         int `json:"combo"`
	// ActivityStatus int `json:"status"`
	MatchID    string     `json:"match_id"`
	CouponFlag bool       `json:"coupons_flag"`
	LCoupons   []*Coupons `json:"love_coupons"`
}

// 活动信息
type ActivityInfo struct {
	Aid             string   `json:"aid"`
	Name            string   `json:"name"`
	SponsorName     string   `json:"sponsor_name"`
	Slogan          string   `json:"slogan"`
	OrgName         string   `json:"org_name"`
	OrgHead         string   `json:"org_head"`
	Creator         string   `json:"creator"`
	StartTime       string   `json:"start_time"`
	EndTime         string   `json:"end_time"`
	RouteId         string   `json:"route_id"`
	Bgpic           string   `json:"bgpic"`
	BgpicStatus     int      `json:"bgpic_status"`
	Type            int      `json:"type"`
	TeamMode        int      `json:"team_mode"`
	TeamOff         int      `json:"team_off"`
	TeamMemberLimit int      `json:"team_member_limit"`
	Status          int      `json:"status"`
	ShowSponsor     int      `json:"show_sponsor"`  // 是否展示发起人信息（适用于移动端发起）, 0表示不展示, 1表示展示发起人信息
	MatchOff        int      `json:"match_off"`     // 是否支持配捐 0表示支持配捐, 1表示不支持配捐, 默认活动支持配捐
	DefaultTeams    []string `json:"default_teams"` // 系统默认小队名称列表
	Rule            string   `json:"rule"`          // 活动规则（pc端发起的活动，可以直接配置）
	Color           string   `json:"color"`         // 项目主色
	CompanyID       string   `json:"company_id"`    // 企业id(运营平台)
	SuppCoupons     bool     `json:"supp_coupons"`  // 是否支持爱心券
	WhiteType       int      `json:"white_type"`    // 白名单类型，0表示不需要手机号码，1表示需要手机号码，2表示要白名单
	WhiteList       []string `json:"white_list"`    // 白名单手机号码数组
	CreateTime      string   `json:"create_time"`   // 创建时间
	Cover           string   `json:"cover"`         // 活动封面图地址
	RelationDesc    string   `json:"relation_desc"` // 活动关联描述(把一些活动关联起来)
	ForwardPic      string   `json:"forward_pic"`   // 转发图片地址
}

// 配捐功能信息（运营平台）
type MatchInfo struct {
	EventId       string `json:"event_id"`
	MatchMode     int    `json:"match_mode"`  // 配捐模式
	RuleType      int    `json:"rule_type"`   // 配捐规则 0表示buff连续配捐规则，1表示buff百分比配捐规则，2表示剩余配捐规则, 3.步数关联配捐
	StepQuota     int    `json:"step_quota"`  // 配捐步数最低门槛
	TargetFund    int64  `json:"target_fund"` // 目标金额，分
	RemindFund    int64  `json:"remind_fund"` // 消息提醒剩余金额
	BasicFund     int    `json:"basic_fund"`
	BasicWave     int    `json:"basic_wave"`
	BuffFund      int    `json:"buff_fund"`
	BuffWave      int    `json:"buff_wave"`
	BuffThreshold int    `json:"buff_threshold"` // 触发buff配捐的阈值
	BuffPercent   int    `json:"buff_percent"`   // buff配捐百分比
	Pid           string `json:"pid"`            // 项目id
	CompanyId     string `json:"company_id"`     // 企业id
	Certificate   string `json:"certificate"`    // 捐赠凭证
	StartTime     string `json:"start_time"`     // 配捐开始时间
	EndTime       string `json:"end_time"`       // 配捐结束时间
	/* 仅当type=3有效 */
	MinSteps int `json:"min_steps"` // 最低步数
	MaxSteps int `json:"max_steps"` // 最大步数
	MinMatch int `json:"min_match"` // 最低配捐
	MaxMatch int `json:"max_match"` // 最大配捐
}

// 配捐功能信息（移动端）
type MatchMobile struct {
	EventId    string `json:"event_id"`
	Pid        string `json:"pid"`
	MatchMode  int    `json:"match_mode"`
	RuleType   int    `json:"rule_type"` // 配捐规则 0表示buff连续配捐规则，1表示buff百分比配捐规则，2表示剩余配捐规则
	StepQuota  int    `json:"step_quota"`
	TargetFund int64  `json:"target_fund"`
}

// 小队信息
type TeamInfo struct {
	TeamId         string `json:"team_id"`
	TeamName       string `json:"team_name"`
	TeamCreator    string `json:"team_creator"`
	TeamLeaderNick string `json:"team_leader_nick"`
	TeamLeaderHead string `json:"team_leader_head"`
	TeamLeaderRank int    `json:"team_leader_rank"`
	TeamRank       int    `json:"team_rank"`
	TeamType       int    `json:"team_type"`
	TeamSteps      int64  `json:"team_steps"`
	TeamMember     int64  `json:"team_member"`
	TeamFlag       int    `json:"team_flag"`
	TeamStaus      int    `json:"team_status"`
	TeamFunds      int64  `json:"team_funds"`
}

// 企业信息
type CompanyInfo struct {
	CompanyId   string `json:"company_id"`
	CompanyName string `json:"company_name"`
	CompanyLogo string `json:"company_logo"`
}

type GetUserCouponsRequest struct {
	Oid     string `json:"oid"`
	OrderNo string `json:"order_no"`
}

type GetUserCouponsResponse struct {
	OrderNo string        `json:"order_no"`
	Pid     string        `json:"pid"`
	Oid     string        `json:"oid"`
	List    []*CouponInfo `json:"list"`
}

type CouponInfo struct {
	CouponId          string `json:"coupon_id"`         // 微信支付发券返回coupon_id
	SendTime          int32  `json:"send_time"`         // 发券时间
	SendResult        int32  `json:"send_result"`       //  发券结果  	0：发放爱心券到微信卡包失败（终态） 	1：发放爱心券到微信卡包成功（终态） 	2：发放中（非终态） 	3：用户已领取过该爱心券（终态）
	CpId              string `json:"cp_id"`             //爱心券ID
	CpType            int32  `json:"cp_type"`           // 1消费券， 2商家券
	CpName            string `json:"cp_name"`           // 爱心券名称
	CpStockType       string `json:"cp_stock_type"`     // 批次类型  [消费券] - NORMAL：固定面额满减券批次 CUT_TO：减至券批次;  [商家券] NORMAL：固定面额满减券批次/ DISCOUNT：折扣券批次/ EXCHANGE：换购券批次
	CpCill            int32  `json:"cp_cill"`           // 批次类型为NORMAL、DISCOUNT、EXCHANGE：消费门槛（单位：分）/ 批次类型为CUT_TO：可用优惠的商品最高单价（单位：分）
	CpReduction       int32  `json:"cp_reduction"`      // 批次类型为NORMAL：优惠金额（单位：分） / 批次类型为DISCOUNT：折扣百分比，例如：86为八六折。示例值：86 / 批次类型为EXCHANGE：单品换购价（单位：分）/ 批次类型为CUT_TO：减至后的优惠单价（单位：分）
	CpScope           string `json:"cp_scope"`          // 可用范围
	CpAppId           string `json:"cp_app_id"`         // 小程序ID
	CpAppletLink      string `json:"cp_applet_link"`    // 券使用的url（商城）
	CpAvailableStart  int32  `json:"cp_vailable_start"` // 使用起始时间
	CpAvailableEnd    int32  `json:"cp_available_end"`  // 使用结束时间
	CpReceiveStart    int32  `json:"cp_receive_start"`  //领取起始时间
	CpReceiveEnd      int32  `json:"cp_receive_end"`    //领取结束时间
	CpMaxPerUser      int32  `json:"cp_max_per_user"`   //用户总领取上限
	CpMaxNum          int32  `json:"cp_max_num"`        //制券数量
	CpSort            int32  `json:"cp_sort"`           //券优先级
	EnterpriseId      string `json:"enterprise_id"`     //企业ID
	EnterpriseName    string `json:"enterprise_name"`
	EnterpriseLogo    string `json:"enterprise_logo"`
	CpDonateThreshold int32  `json:"cp_donate_threshold"` //捐款门槛
	CpAppOriginalId   string `json:"cp_app_original_id"`  // 原始应用ID
}

type GetActivityBaseRequest struct {
	Aid string `json:"aid"`
	Oid string `json:"oid"`
}

type GetActivityBaseResponse struct {
	Activity ActivityInfo `json:"activity"`
}

// 查询用户丰巢券领取结果
type GetUserFengchaoCouponsRequest struct {
	Oid   string `json:"oid"`
	ActId string `json:"act_id"`
	Code  string `json:"code"`
}
type GetUserFengchaoCouponsResponse struct {
	CouponsStatus int `json:"coupons_status"`
}

// 用户领取丰巢券
type ReceiveFengchaoCouponsRequest struct {
	UniId         string `json:"uni_id"`
	Oid           string `json:"oid"`
	ActId         string `json:"act_id"`
	Code          string `json:"code"`
	Phone         string `json:"phone"`
	RealPhone     string `json:"-"`
	EncryptedData string `json:"encrypted_data"`
	EncryptedIV   string `json:"encrypted_iv"`
	OrderId       string `json:"order_id"`
}

type ReceiveFengchaoCouponsResponse struct {
	Status int    `json:"status"`
	Attach string `json:"attach"`
}

type GetActivityRequest struct {
	Aid string `json:"aid"`
	Oid string `json:"oid"`
}

// 活动型证书
type Cert struct {
	OrgId        string `json:"org_id"`        // 机构id
	OrgName      string `json:"org_name"`      // 机构名称
	OrgSeal      string `json:"org_seal"`      // 机构印章
	SerialNumber string `json:"serial_number"` // 证书编号
}

// 查询活动信息
type GetActivityResponse struct {
	Activity       ActivityInfo `json:"activity"`
	Team           TeamInfo     `json:"team"`
	Company        CompanyInfo  `json:"company"`
	Head           string       `json:"head"`
	Nick           string       `json:"nick"`
	Join           bool         `json:"join"`
	IsCreator      bool         `json:"is_creator"`
	UserRank       int64        `json:"user_rank"`
	ActivityMember int64        `json:"activity_member"`
	ActivitySteps  int64        `json:"activity_steps"`
	TeamCnt        int64        `json:"team_cnt"`
	TeamSteps      int64        `json:"team_steps"`
	TodaySteps     int64        `json:"today_steps"`
	TotalSteps     int64        `json:"total_steps"`
	MatchMoney     int64        `json:"match_money"`  // 用户的活动配捐金额, 单位分
	MatchSteps     int64        `json:"match_steps"`  // 用户的活动捐出步数
	SponsorHead    string       `json:"sponsor_head"` // 发起人头像（移动端活动）
	SponsorNick    string       `json:"sponsor_nick"` // 发起人昵称（移动端活动）
	Cert           Cert         `json:"cert"`         // 活动型证书
	Class          int64        `json:"class"`        // 活动类别标识 // 0:普通类型（default）  1：支持公益金排行
}

// 创建活动（运营平台）
type CreateActivityPlatformRequest struct {
	Token    string       `json:"token"` // 创建活动token（用于运营平台发起活动）
	Oid      string       `json:"oid"`
	Appid    string       `json:"appid"`
	UniId    string       `json:"uni_id"`
	Activity ActivityInfo `json:"activity"` // 活动配置
	Match    MatchInfo    `json:"match"`    // 配捐功能（运营平台）
}

type CreateActivityPlatformResponse struct {
	Aid string `json:"aid"`
}

// 创建活动（用户移动端）
type CreateActivityMobileRequest struct {
	Oid         string       `json:"oid"`
	Appid       string       `json:"appid"`
	UniId       string       `json:"uni_id"`
	Activity    ActivityInfo `json:"activity"` // 活动配置
	MatchMobile MatchMobile  `json:"match"`    // 配捐功能（移动端）
}

type CreateActivityMobileResponse struct {
	Aid string `json:"aid"`
}

// 创建小队
type CreateTeamRequest struct {
	Oid   string   `json:"oid"`
	Appid string   `json:"appid"`
	UniId string   `json:"uni_id"`
	Aid   string   `json:"aid"`
	Team  TeamInfo `json:"team"` // 小队配置
	Token string   `json:"token"`
}

type CreateTeamResponse struct {
	Aid    string `json:"aid"`
	TeamId string `json:"team_id"`
}

// 加入/退出活动或小队
type JoinActivityRequest struct {
	UniId         string `json:"uni_id"`
	Appid         string `json:"appid"`
	Oid           string `json:"oid"`
	Aid           string `json:"aid"`
	TeamID        string `json:"team_id"`
	Join          bool   `json:"join"`
	EncryptedData string `json:"encrypted_data"`
	EncryptedIV   string `json:"encrypted_iv"`
	Code          string `json:"code"`
	PhoneNum      string `json:"-"`
}

type JoinRandTeamResponse struct {
	TeamID   string `json:"team_id"`
	TeamName string `json:"team_name"`
}

type JoinActivityResponse struct {
	JoinRandTeamResponse
}

// 查询活动小队赛道排名
type GetActivityTeamRankRequest struct {
	Aid      string `json:"aid"`
	Page     int64  `json:"page"`
	Size     int64  `json:"size"`
	RankType string `json:"rank_type"` // step:步数排名  fund：公益金排名
	Oid      string `json:"oid"`
}

type ActivityTeamRank struct {
	Aid        string `json:"aid"`
	TeamId     string `json:"team_id"`
	Rank       int64  `json:"rank"`
	Name       string `json:"name"`
	Head       string `json:"head"`
	TotalUsers int64  `json:"total_users"`
	TotalSteps int64  `json:"total_steps"`
	TotalFunds int64  `json:"total_funds"`
}

type GetActivityTeamRankResponse struct {
	UpdateTime string             `json:"update_time"`
	Total      int64              `json:"total"`
	List       []ActivityTeamRank `json:"list"`
	TeamRank   int64              `json:"team_rank"`
}

// 查询活动个人赛道排名  //
type GetActivityUserRankRequest struct {
	Aid      string `json:"aid"`
	Oid      string `json:"oid"`
	Page     int64  `json:"page"`
	Size     int64  `json:"size"`
	Token    string `json:"token"`     // 适用于运营平台, 移动端忽略该参数
	RankType string `json:"rank_type"` // step:步数排名  fund：公益金排名
}

type ActivityUserRank struct {
	Oid         string `json:"oid"`
	Rank        int64  `json:"rank"`
	Nick        string `json:"nick"`
	Head        string `json:"head"`
	TotalSteps  int64  `json:"total_steps"`
	TodaySteps  int64  `json:"today_steps"`
	DonateMoney int64  `json:"donate_money"`
	IsSelf      bool   `json:"is_self"`
	TeamName    string `json:"team_name"`
	TeamID      string `json:"team_id"`
	EncryptOid  string `json:"encrypt_oid"`
	TotalFunds  int64  `json:"total_funds"`
}

type ActivityUserRankPlatform struct {
	Nick       string `json:"nick"`
	Head       string `json:"head"`
	EncryptOid string `json:"encrypt_oid"`
	CreateTime string `json:"create_time"`
}

type GetActivityUserRankResponse struct {
	UpdateTime string             `json:"update_time"`
	Total      int64              `json:"total"`
	List       []ActivityUserRank `json:"list"`
	UserRank   int64              `json:"user_rank"`
}

type GetActivityUserRankPlatformResponse struct {
	UpdateTime string                     `json:"update_time"`
	Total      int64                      `json:"total"`
	List       []ActivityUserRankPlatform `json:"list"`
}

type GetTeamsRequest struct {
	Aid     string   `json:"aid"`
	Oid     string   `json:"oid"`
	TeamIds []string `json:"team_ids"`
}

type GetTxStaffIdentityRequest struct {
	RtxName string `json:"name"`
	Oid     string `json:"oid"`
}

type StaffInfo struct {
	StaffName                      string `json:"StaffName"`
	StaffTypeID                    int64  `json:"StaffTypeID"`
	StaffTypeName                  string `json:"StaffTypeName"`
	EntryYearsID                   int64  `json:"EntryYearsID"`
	EntryYearsName                 string `json:"EntryYearsName"`
	CharityGoalStepCount           int64  `json:"CharityGoalStepCount"`
	CharityGoalTree                int64  `json:"CharityGoalTree"`
	CharityGoalName                string `json:"CharityGoalName"`
	AnniversaryOfEmploymentRanking int64  `json:"AnniversaryOfEmploymentRanking"`
}

type GetTxStaffIdentityRespFail struct {
	GetTxStaffIdentityRespHead
	Data string `json:"data"` //
}

type GetTxStaffIdentityRespSucc struct {
	GetTxStaffIdentityRespHead
	Data *StaffInfo `json:"data"` //
}

type GetTxStaffIdentityRespHead struct {
	SCode       int64 `json:"statuscode"`  //
	IsEncrypted bool  `json:"IsEncrypted"` //
}

type GetTxStaffIdentityResponse struct {
	StaffName   string `json:"staff_name"`   // 员工姓名
	Oid         string `json:"oid"`          // 微信openid
	IsStaff     int64  `json:"is_staff"`     // 是否是合法员工  1：是 0：否
	StaffType   int64  `json:"staff_type"`   // 员工类型 1.为正式员工、2.为非正式员工
	WorkYears   int64  `json:"work_years"`   // 服务年限 ：1-22为1到22周年的员工、98为无司龄员工（如：非正式员工）、99为不满一周年员工
	TargetSteps int64  `json:"target_steps"` //目标步数
	TargetTrees int64  `json:"target_trees"` //目标颗数
	TargetDesc  string `json:"target_desc"`  //目标描述："走完28000步，种下3棵胡杨树。",
	ActRanking  int64  `json:"act_ranking"`  //司庆活动排位，第102位
}

type Team struct {
	TeamInfo
	IsWatch bool `json:"is_watch"`
}

// 查询小队信息
type GetTeamsResponse struct {
	Teams []Team `json:"teams"`
}

// 查询小队成员排名
type GetTeamUserRankRequest struct {
	Aid    string `json:"aid"`
	Oid    string `json:"oid"`
	TeamID string `json:"team_id"`
	Page   int64  `json:"page"`
	Size   int64  `json:"size"`
}

type TeamUserRank struct {
	Oid          string `json:"oid"`
	Rank         int64  `json:"rank"`
	Nick         string `json:"nick"`
	Head         string `json:"head"`
	TotalSteps   int64  `json:"total_steps"`
	TodaySteps   int64  `json:"today_steps"`
	DonateMoney  int64  `json:"donate_money"`
	IsSelf       bool   `json:"is_self"`
	IsTeamLeader bool   `json:"is_team_leader"`
}

type GetTeamUserRankResponse struct {
	UpdateTime   string         `json:"update_time"`
	Total        int64          `json:"total"`
	IsWatch      bool           `json:"is_watch"`
	IsTeamLeader bool           `json:"is_team_leader"`
	List         []TeamUserRank `json:"list"`
}

// 活动统计信息
type ActivityStats struct {
	Aid             string `json:"aid"`
	Name            string `json:"name"`
	SponsorName     string `json:"sponsor_name"`
	Bgpic           string `json:"bgpic"`
	BgpicStatus     int    `json:"bgpic_status"`
	Type            int    `json:"type"`
	TeamMode        int    `json:"team_mode"`
	Status          int    `json:"status"`
	MatchName       string `json:"match_name"`
	MatchLogo       string `json:"match_logo"`
	TeamCnt         int64  `json:"team_cnt"`          // 小队数量
	ActivityMember  int64  `json:"activity_member"`   // 活动成员数量
	ActivitySteps   int64  `json:"activity_steps"`    // 活动总步数
	MatchOff        int    `json:"match_off"`         // 是否支持配捐 0表示支持配捐, 1表示不支持配捐, 默认活动支持配捐
	MatchRemainFund int64  `json:"match_remain_fund"` // 活动今天剩余配捐金额, 前面未配完的金额会累计到今天, 单位分
	ShowSponsor     int    `json:"show_sponsor"`      // 是否展示发起人信息(适用于移动端发起), 1表示展示发起人信息
	Color           string `json:"color"`             // 项目主色
	Cover           string `json:"cover"`             // 活动封面图地址
	RelationDesc    string `json:"relation_desc"`     // 活动关联描述(把一些活动关联起来)
}

// 查询用户参与的活动列表
type GetUserJoinActivityRequest struct {
	Oid  string `json:"oid"`
	Page int64  `json:"page"`
	Size int64  `json:"size"`
	Type int64  `json:"type"`
}

type GetUserJoinActivityResponse struct {
	Total     int64           `json:"total"`
	HasCreate bool            `json:"has_create"` // 是否成功创建过活动型
	List      []ActivityStats `json:"list"`
}

// 查询活动
type GetActivityListRequest struct {
	Token      string `json:"token"`
	Offset     int32  `json:"offset"`
	Size       int32  `json:"size"`
	ActivityID string `json:"activity_id"`
	Status     int32  `json:"status"`
	Name       string `json:"name"`
}

type ActivityItem struct {
	Activity ActivityInfo `json:"activity"`
	Match    MatchInfo    `json:"match"`
}

type GetActivityListResponse struct {
	Total int32          `json:"total"`
	Items []ActivityItem `json:"items"`
}

// 查询活动小队
type GetActivityTeamListRequest struct {
	Token      string `json:"token"`
	ActivityID string `json:"activityID"`
	Offset     int32  `json:"offset"`
	Size       int32  `json:"size"`
	Type       int32  `json:"type"`
}

type GetActivityTeamListResponse struct {
	Total int32      `json:"total"`
	Items []TeamInfo `json:"items"`
}

// 修改活动（移动端）
type UpdateActivityMobileRequest struct {
	Oid      string       `json:"oid"`      // 用户id
	Aid      string       `json:"aid"`      // 活动id
	Activity ActivityInfo `json:"activity"` // 活动配置
	Match    MatchInfo    `json:"match"`    // 配捐配置
}

type UpdateActivityMobileResponse struct {
	Result   bool `json:"result"`
	RespCode int  `json:"resp_code"`
}

// 修改活动（运营平台）
type UpdateActivityRequest struct {
	Token      string       `json:"token"`
	ActivityID string       `json:"activityID"`
	Activity   ActivityInfo `json:"activity"` // 活动配置
	Match      MatchInfo    `json:"match"`    // 配捐功能（运营平台）
}

type UpdateActivityResponse struct {
	Result bool `json:"result"`
}

type UpdateActivityStateRequest struct {
	Token      string `json:"token"`
	ActivityID string `json:"activityID"`
	State      int32  `json:"state"`
}

type UpdateActivityStateResponse struct {
	Result bool `json:"result"`
}

// 修改小队
type UpdateActivityTeamRequest struct {
	Token      string `json:"token"`
	ActivityID string `json:"activityID"`
	TeamID     string `json:"teamID"`
	TeamName   string `json:"team_name"`
}

type UpdateActivityTeamResponse struct {
	Result bool `json:"result"`
}

// 删除活动
type DeleteActivityRequest struct {
	Token      string `json:"token"`
	ActivityID string `json:"activityID"`
	Operator   string `json:"oid"`
}

type DeleteActivityResponse struct {
	Result bool `json:"result"`
}

// 删除小队
type DeleteActivityTeamRequest struct {
	Token      string `json:"token"`
	ActivityID string `json:"activityID"`
	TeamID     string `json:"teamID"`
}

type DeleteActivityTeamResponse struct {
	Result bool `json:"result"`
}

type GetActivityReportRequest struct {
	Week string `json:"week"`
	Oid  string `json:"oid"`
	Page int32  `json:"page"`
	Size int32  `json:"size"`
}

type ActivityReport struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      int32  `json:"status"`
	TeamMode    int32  `json:"team_mode"`
	IsMatch     bool   `json:"is_match"`
	RouteID     string `json:"route_id"`
	Bgpic       string `json:"bgpic"`
	BgpicStatus int32  `json:"bgpic_status"`
	UserRank    int64  `json:"user_rank"`
	TeamRank    int64  `json:"team_rank"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

type GetActivityReportResponse struct {
	List []ActivityReport `json:"list"`
}

type ModifyTeamRequest struct {
	Oid        string `json:"oid"`
	ActivityID string `json:"activity_id"`
	TeamID     string `json:"team_id"`
	TeamDesc   string `json:"team_desc"`
	TeamType   int32  `json:"team_type"`
}

type ModifyTeamResponse struct {
	TeamID string `json:"team_id"`
}

type GetTeamUserRequest struct {
	Oid        string `json:"oid"`
	ActivityID string `json:"activity_id"`
	TeamID     string `json:"team_id"`
	Page       int32  `json:"page"`
	Size       int32  `json:"size"`
}

type TeamUser struct {
	Oid    string `json:"oid"`
	Nick   string `json:"nick"`
	Head   string `json:"head"`
	IsSelf bool   `json:"is_self"`
}

type GetTeamUserResponse struct {
	List []TeamUser `json:"list"`
}

type RemoveTeamUserRequest struct {
	Oid         string `json:"oid"`
	ActivityID  string `json:"activity_id"`
	TeamID      string `json:"team_id"`
	OperatorOid string `json:"operator_oid"`
}

type RemoveTeamUserResponse struct {
}

// 活动地图头像展示
type GetActivityMapHeadRequest struct {
	Oid        string `json:"oid"`
	ActivityID string `json:"activity_id"`
}

type ActivityMapHeadLead struct {
	Nick   string `json:"nick"`
	Head   string `json:"head"`
	Type   int32  `json:"type"`
	Step   int64  `json:"step"`
	Rank   int64  `json:"rank"`
	IsSelf bool   `json:"is_self"`
}

type ActivityMapHeadPoint struct {
	Interval int64 `json:"interval"`
	Num      int64 `json:"num"`
}

type GetActivityMapHeadResponse struct {
	Leads  []ActivityMapHeadLead  `json:"leads"`
	Points []ActivityMapHeadPoint `json:"points"`
}

// 小队地图头像展示
type GetTeamMapHeadRequest struct {
	Oid        string `json:"oid"`
	ActivityID string `json:"activity_id"`
	TeamID     string `json:"team_id"`
}

type TeamMapHeadLead struct {
	Nick   string `json:"nick"`
	Head   string `json:"head"`
	Type   int32  `json:"type"`
	Step   int64  `json:"step"`
	Rank   int64  `json:"rank"`
	IsSelf bool   `json:"is_self"`
}

type TeamMapHeadPoint struct {
	Interval int64 `json:"interval"`
	Num      int64 `json:"num"`
}

type GetTeamMapHeadResponse struct {
	Leads  []TeamMapHeadLead  `json:"leads"`
	Points []TeamMapHeadPoint `json:"points"`
}

type GetActivityBattleReportRequest struct {
	Aid string `json:"aid"`
}

type GetActivityBattleReportResponse struct {
	Name         string `json:"name"`
	Slogan       string `json:"slogan"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Type         int32  `json:"type"`
	SponsorHead  string `json:"sponsor_head"`
	SponsorNick  string `json:"sponsor_nick"`
	Rule         string `json:"rule"`
	Pid          string `json:"pid"`
	ProjName     string `json:"proj_name"`
	TeamMode     int32  `json:"team_mode"`
	MatchOff     int32  `json:"match_off"`     // 是否支持配捐 0表示支持配捐, 1表示不支持配捐, 默认活动支持配捐
	ShowSponsor  int32  `json:"show_sponsor"`  // 是否展示发起人信息（适用于移动端发起）, 0表示不展示, 1表示展示发起人信息
	RelationDesc string `json:"relation_desc"` // 活动关联描述(把一些活动关联起来)
}

type GetActivityMatchRankRequest struct {
	Aid  string `json:"aid"`
	Page int32  `json:"page"`
	Size int32  `json:"size"`
}

type ActivityMatchRank struct {
	Nick     string `json:"nick"`
	Head     string `json:"head"`
	Rank     int32  `json:"rank"`
	Money    int32  `json:"money"`
	TeamID   string `json:"team_id"`
	TeamName string `json:"team_name"`
}

type GetActivityMatchRankResponse struct {
	List []ActivityMatchRank `json:"list"`
}

type ChangeTeamLeaderRequest struct {
	Oid         string `json:"oid"`
	ActivityID  string `json:"activity_id"`
	TeamID      string `json:"team_id"`
	OperatorOid string `json:"operator_oid"`
}

type ChangeTeamLeaderResponse struct {
}

type GetActivityMatchInfoRequest struct {
	Aid string `json:"activity_id"`
}

type GetActivityMatchInfoResponse struct {
	Exist bool `json:"exist"`

	PID    string `json:"project_id"`
	PTitle string `json:"project_title"`
	PImg   string `json:"project_img"`
	PExt   string `json:"ext_donate"`

	CID       string `json:"company_id"`
	CShotName string `json:"company_short"`
	CFullName string `json:"company_full"`
	CLogo     string `json:"company_logo"`

	MatchItem string `json:"match_item"`
	Status    int    `json:"status"`
	Quota     int    `json:"quota"`
	Mode      int    `json:"mode"`

	TargetFund int64 `json:"target_fund"`
	RemainFund int64 `json:"remind_fund"`
	MatchFund  int64 `json:"match_fund"`
	MatchCnt   int   `json:"match_cnt"`
}

// 删除活动成员
type DeleteActivityUserRequest struct {
	Aid        string `json:"aid"`
	Oid        string `json:"oid"`
	EncryptOid string `json:"encrypt_oid"` // 适用于运营平台, aes加密的用户openid,
	Token      string `json:"token"`       // 适用于运营平台, 移动端忽略该参数
}

type DeleteActivityUserResponse struct {
	Result bool `json:"result"` // true表示成功
}

type GetYQZCKVMapRequest struct {
	RouteID string `json:"route_id"`
}

type GetYQZCKVMapResponse struct {
	Result  bool   `json:"result"`
	Message string `json:"message"`
}

type GetYQZMapSummaryRequest struct {
}

type YQZMapSummary struct {
	WeekID   string `json:"week_id"`
	RouteID  int    `json:"route_id"`
	Week     int    `json:"week"`
	Name     string `json:"name"`
	Abbr     string `json:"abbr"`
	Tag      string `json:"tag"`
	Cover    string `json:"cover"`
	Distance int    `json:"distance"`
	Day      int    `json:"day"`
	UserNum  int64  `json:"user_num"`
}

type GetYQZMapSummaryResponse struct {
	MapData YQZMapSummary `json:"map_data"`
}

type GetYQZMapDetailRequest struct {
	Oid string `json:"oid"`
	// poi推送数据
	IsMsgPush bool   `json:"is_msg_push"`
	ArriveNum int    `json:"arrive_num"`
	MsgWeekID string `json:"msg_week_id"`
	// 配捐推送
	IsMatchPush bool   `json:"is_match_push"`
	MatchDate   string `json:"match_date"`
}

type ShowPoint struct {
	Oid      string  `json:"oid"`
	Head     string  `json:"head"`
	Walked   float64 `json:"walked"` // km, if not use we abort
	Step     int64   `json:"step"`
	Rank     int     `json:"rank"` // 0 mean no rank
	IsMyself bool    `json:"is_myself"`
	// IsBlack  bool    `json:"is_black"`
}

type IntervalStat struct {
	Interval int   `json:"interval"`
	Num      int64 `json:"num"`
	Percent  int   `json:"percent"`
}

type GetYQZMapDetailResponse struct {
	//IsDonate         bool            `json:"is_donate"`
	MapData           YQZMapSummary   `json:"map_data"`
	MapShowPoints     []*ShowPoint    `json:"map_show_points"`
	MapIntervalStats  []*IntervalStat `json:"map_interval_stats"`
	MapIntervalColour []*IntervalStat `json:"map_interval_colour"`
}

type GetYQZUserProfileRequest struct {
	Oid    string `json:"oid"`
	UninId string `json:"unin_id"`
}

type YQZStepInfo struct {
	TodaySteps int32       `json:"today_steps"`
	WeekStep   []int32     `json:"week_steps"`
	MonthStep  []*DateStep `json:"month_steps"`
	UpdateTime string      `json:"update_time"`
	TodayDate  string      `json:"today_date"`
}

func (m DateStep) String() string {
	return fmt.Sprintf("{date: %v, step: %v}", m.Date, m.Step)
}

type YQZMatchInfo struct {
	TotalFunds int64 `json:"total_funds"`
	TotalTimes int   `json:"total_times"`
	TotalSteps int   `json:"total_steps"`
}

type YQZSpecialInfo struct {
	IsDonate      bool   `json:"is_donate"`
	IsJoin        bool   `json:"is_join"`
	IsAuto        bool   `json:"is_auto"`
	Leaf          int    `json:"leaf"`
	InCurrentYQZ  bool   `json:"in_current_yqz"`
	DefeatPercent int    `json:"defeat_percent"`
	PKObject      string `json:"pk_object"`
}

type GetYQZUserProfileResponse struct {
	Nick            string          `json:"nick"`
	Head            string          `json:"head"`
	UserStepInfo    *YQZStepInfo    `json:"step_info"`
	UserMatchInfo   *YQZMatchInfo   `json:"match_info"`
	UserSpecialInfo *YQZSpecialInfo `json:"special_info"`
}

func (m GetYQZUserProfileResponse) String() string {
	return fmt.Sprintf("{nick: %v, head: %v, step:{today: %v, week: %v, month: %v}, fund:{total: %v, times: %v}, other:{donate: %v, join: %v, auto: %v, leaf: %v, in: %v, defeat: %v}}", m.Nick, m.Head, m.UserStepInfo.TodayDate, m.UserStepInfo.WeekStep, m.UserStepInfo.MonthStep, m.UserMatchInfo.TotalFunds, m.UserMatchInfo.TotalTimes, m.UserSpecialInfo.IsDonate, m.UserSpecialInfo.IsJoin, m.UserSpecialInfo.IsAuto, m.UserSpecialInfo.Leaf, m.UserSpecialInfo.InCurrentYQZ, m.UserSpecialInfo.DefeatPercent)
}

type GetYQZTeamRequest struct {
	Oid    string `json:"oid"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
}

type YQZTeamUser struct {
	Nick string `json:"nick"`
	Head string `json:"head"`
	Step int32  `json:"step"`
}

type YQZTeamInfo struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	TotalUsers int32          `json:"total_users"`
	WeekID     string         `json:"week_id"`
	RouteId    int32          `json:"route_id"`
	Rank       int32          `json:"rank"`
	TeamType   int32          `json:"team_type"` // animal type(1、蜗牛 2、马 3、老鹰)
	TopSize    int32          `json:"top_user_size"`
	TopUser    []*YQZTeamUser `json:"top_user"` // people finish
	LastSize   int32          `json:"last_user_size"`
	LastUser   []*YQZTeamUser `json:"last_user"` // people not start
	LeafSize   int32          `json:"leaf_user_size"`
	LeafUser   []*YQZTeamUser `json:"leaf_user"` // people send leaf
}

type GetYQZTeamResponse struct {
	Total int            `json:"total"`
	Teams []*YQZTeamInfo `json:"teams"`
}

// 司庆活动用户证书信息
type GetStaffCertificateRequest struct {
	StaffName string `json:"staff_name"`
}

type GetStaffCertificateResponse struct {
	IsFinish   bool   `json:"is_finish"`
	CertNumber string `json:"cert_number"`
	Nick       string `json:"nick"`
	Steps      int64  `json:"steps"`
}

// 查询司庆活动个人赛道排名
type GetStaffUserRankRequest struct {
	Oid  string `json:"oid"`
	Aid  string `json:"aid"`
	Page int64  `json:"page"`
	Size int64  `json:"size"`
}

type StaffUserRank struct {
	Rank        int64  `json:"rank"`
	Nick        string `json:"nick"`
	Head        string `json:"head"`
	TotalSteps  int64  `json:"total_steps"`
	TodaySteps  int64  `json:"today_steps"`
	DonateMoney int64  `json:"donate_money"`
	IsSelf      bool   `json:"is_self"`
	TeamName    string `json:"team_name"`
	TeamID      string `json:"team_id"`
	EncryptOid  string `json:"encrypt_oid"`
	IsFinish    bool   `json:"is_finish"`
	TargetSteps int64  `json:"target_steps"`
	TargetTrees int64  `json:"target_trees"`
}

type GetStaffUserRankResponse struct {
	UpdateTime string          `json:"update_time"`
	Total      int64           `json:"total"`
	List       []StaffUserRank `json:"list"`
}

type GetSponsorCountRequest struct {
}

type GetSponsorCountResponse struct {
	Count int `json:"count"`
}

type GetCompanyMatchRankRequest struct {
	Offset int `json:"offset"`
	Size   int `json:"size"`
}

type GetCompanyMatchRankResponse struct {
	TotalCompanies int64              `json:"total_companies"`
	TotalUsers     int64              `json:"total_users"`
	TotalFund      int64              `json:"total_fund"`
	StatDate       string             `json:"stat_date"`
	CompanyRank    []*CompanyRankInfo `json:"company_rank"`
}

type CompanyRankInfo struct {
	Pic  string `json:"pic"`
	Name string `json:"name"`
	Fund int64  `json:"fund"`
}

// 活动成员黑名单
type BlacklistActivityUserPlatformRequest struct {
	Aid        string `json:"aid"`
	Oid        string `json:"oid"`
	EncryptOid string `json:"encrypt_oid"` // 适用于运营平台, aes加密的用户openid,
	Token      string `json:"token"`       // 适用于运营平台, 移动端忽略该参数
	IsBlack    bool   `json:"is_black"`    // 是否设置成黑名单
}

type BlacklistActivityUserPlatformResponse struct {
	Result bool `json:"result"` // true表示成功
}

// 活动成员黑名单列表
type BlacklistActivityUserListPlatformRequest struct {
	Aid   string `json:"aid"`
	Token string `json:"token"` // 适用于运营平台, 移动端忽略该参数
	Page  int32  `json:"page"`
	Size  int32  `json:"size"`
}

type Blacklist struct {
	EncryptOid string `json:"encrypt_oid"` // 适用于运营平台, aes加密的用户openid,
	Nick       string `json:"nick"`
	Head       string `json:"head"`
}

type BlacklistActivityUserListPlatformResponse struct {
	List []Blacklist `json:"list"`
}

type GetUsersStepsRequest struct {
	Oids        []string `json:"oids"`
	EncryptOids []string `json:"encrypt_oids"`
}

type UserStepsInfo struct {
	Oid          string       `json:"oid"`
	Nick         string       `json:"nick"`
	Head         string       `json:"head"`
	UserStepInfo *YQZStepInfo `json:"step_info"`
}

func (m UserStepsInfo) String() string {
	return fmt.Sprintf("{oid: %v, nick: %v, head: %v, step:{today: %v, week: %v, month: %v}}", m.Oid, m.Nick, m.Head, m.UserStepInfo.TodayDate, m.UserStepInfo.WeekStep, m.UserStepInfo.MonthStep)
}

type GetUsersStepsResponse struct {
	StepsInfo []*UserStepsInfo `json:"users_steps_info"`
}

type GetPKCandidatesRequest struct {
	Oid string `json:"oid"`
}

type GetPKCandidatesResponse struct {
	Candidates []*UserInfo `json:"candidates"`
}

type SetUserPKObjRequest struct {
	Oid string `json:"oid"`
	Obj string `json:"obj"`
}

type SetUserPKObjResponse struct {
	Result bool `json:"result"`
}

type GetUserEncryptIDRequest struct {
	Oid string `json:"oid"`
}

type GetUserEncryptIDResponse struct {
	EncryptID string `json:"encrypt_id"`
}

type GetYqzUsersPkProfileRequest struct {
	Oids        []string `json:"oids"`
	EncryptOids []string `json:"encrypt_oids"`
}

type GetYqzUsersPkProfileResponse struct {
	Profiles []PkProfile `json:"pk_profiles"`
}

type PkProfile struct {
	Oid   string `json:"oid"`
	Thumb int    `json:"thumb"`
	Smile int    `json:"smile"`
	Bomb  int    `json:"bomb"`
}

type GetYqzPkInteractRequest struct {
	Oid   string `json:"oid"`
	ToOid string `json:"to_oid"`
}

type GetYqzPkInteractResponse struct {
	//Ops int `json:"ops"` // 0: not interact/ 1: thumb/2: smile/3: bomb
	//Profile PkProfile `json:"pk_profile"`
	Thumb int `json:"thumb"`
	Smile int `json:"smile"`
	Bomb  int `json:"bomb"`
}

type SetPkInteractRequest struct {
	Oid      string `json:"oid"`
	ToOid    string `json:"to_oid"`
	Interact int    `json:"interact"` // 1: thumb/ 2: smile/ 3: bomb
	Ops      int    `json:"ops"`      // 1: plus/ 2: minus
}

type SetPkInteractResponse struct {
	Result int `json:"result"` // 0: success/ -1: failed/ 1: already set
}

type SetActivePkRequest struct {
	Oid   string `json:"oid"`
	ToOid string `json:"to_oid"`
}

type SetActivePkResponse struct {
	Result int `json:"result"` // 0: success/ -1: failed
}

type SetResponsePkRequest struct {
	Oid   string `json:"oid"`
	ToOid string `json:"to_oid"`
}

type SetResponsePkResponse struct {
	Result int `json:"result"` // 0: success/ -1: failed
}

type PkHistory struct {
	Thumb    int      `json:"thumb"`
	Smile    int      `json:"smile"`
	Bomb     int      `json:"bomb"`
	Date     string   `json:"date"`
	Self     UserInfo `json:"self"`
	Opponent UserInfo `json:"opponent"`
}

func (m *PkHistory) String() string {
	return fmt.Sprintf("{self: %v, opponent: %v, date: %v, thumb: %v, smile: %v, bomb: %v }", m.Self.Nick, m.Opponent.Nick, m.Date, m.Thumb, m.Smile, m.Bomb)
}

type GetPassivePkRequest struct {
	Oid string `json:"oid"`
}

type GetPassivePkResponse struct {
	Result  int          `json:"result"` // 0: success/ -1: failed
	History []*PkHistory `json:"history"`
}

type GetPkNotificationRequest struct {
	Oid string `json:"oid"`
}

type GetPkNotificationResponse struct {
	Result  int          `json:"result"` // 0: success/ -1: failed
	History []*PkHistory `json:"history"`
}
