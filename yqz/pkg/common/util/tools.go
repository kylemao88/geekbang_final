//

package util

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"git.code.oa.com/gongyi/go_common_v2/util"
	"git.code.oa.com/gongyi/gongyi_base/component/gy_seq"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var Loc *time.Location

func init() {
	Loc, _ = time.LoadLocation("Asia/Shanghai")
}

// GetCallee return the callee function name
func GetCallee() string {
	pc, _, _, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash
	return funcName[lastDot+1:]
}

// GetCaller return the function's caller name
func GetCaller() string {
	pc, _, _, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash
	return funcName[lastDot+1:]
}

// GetUsedTime : return time the function used
// demo: defer GetUsedTime(functionName)()
func GetUsedTime(msg string) func() {
	start := time.Now()
	logger.Info("GetUsedTime - enter %s", msg)
	return func() {
		logger.Info("GetUsedTime - exit %s (%s)", msg, time.Since(start))
	}
}

// PrintStack : print stack
func PrintStack() []byte {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	return buf[:n]
}

// GetGoID get goroutine id
func GetGoID() int64 {
	var (
		buf [64]byte
		n   = runtime.Stack(buf[:], false)
		stk = strings.TrimPrefix(string(buf[:n]), "goroutine ")
	)

	idField := strings.Fields(stk)[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Errorf("can not get goroutine id: %v", err))
	}

	return int64(id)
}

// GetFuncPointer get function pointer
func GetFuncPointer(f interface{}) uintptr {
	if f == nil {
		return 0
	}
	return reflect.ValueOf(f).Pointer()
}

func TimeSubDays(start_time, end_time time.Time) int {
	start_time = time.Date(start_time.Year(), start_time.Month(), start_time.Day(), 0, 0, 0, 0, time.Local)
	end_time = time.Date(end_time.Year(), end_time.Month(), end_time.Day(), 0, 0, 0, 0, time.Local)

	return int(end_time.Sub(start_time).Hours() / 24)
}

// GetLocalTime get shanghai time
func GetLocalTime() time.Time {
	return time.Now().In(Loc)
}

// GetLocalFormatTime get format shanghai time
func GetLocalFormatTime() string {
	return GetLocalTime().Format("2006-01-02 15:04:05")
}

// GetLocalFormatDate get format shanghai date
func GetLocalFormatDate() string {
	return GetLocalTime().Format("2006-01-02")
}

// GetLocalDeltaFormatDate get delta format shanghai date
func GetLocalDeltaFormatDate(delta string) (string, error) {
	duration, err := time.ParseDuration(delta)
	if err != nil {
		return "", err
	}
	deltaDate := time.Now().Add(duration).In(Loc).Format("2006-01-02")
	return deltaDate, nil
}

// GetLocalDeltaFormatTime get delta format shanghai date
func GetLocalDeltaFormatTime(delta string) (int64, error) {
	duration, err := time.ParseDuration(delta)
	if err != nil {
		return 0, err
	}
	deltaTime := time.Now().Add(duration).In(Loc).Unix()
	return deltaTime, nil
}

func Datetime2Date(dateTime string) string {
	d, _ := time.ParseInLocation("2006-01-02 15:04:05", dateTime, Loc)
	return d.Format("2006-01-02")
}

func Datetime2Timestamp(dateTime string) int64 {
	d, _ := time.ParseInLocation("2006-01-02 15:04:05", dateTime, Loc)
	return d.Unix()
}

// FormatTime format time
func FormatTime(timestamp int64) string {
	tm := time.Unix(int64(timestamp), 0)
	return tm.In(Loc).Format("2006-01-02 15:04:05")
}

// FormatDate format date
func FormatDate(timestamp int64) string {
	tm := time.Unix(int64(timestamp), 0)
	return tm.In(Loc).Format("2006-01-02")
}

// FormatTime format time
func FormatTimestampToStruct(timestamp int64) time.Time {
	tm := time.Unix(int64(timestamp), 0)
	return tm
}

// DeepCopy is used to copy struct
func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

// WeekStartEnd ...
func WeekStartEnd(t time.Time) (time.Time, time.Time) {
	offset := int(time.Monday - t.Weekday())
	if offset > 0 {
		offset = -6
	}

	weekStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	return weekStart, weekEnd
}

func GetTransId(prefix string) (transid string, err error) {
	labale, seq, err := gy_seq.GetLabelSeq(0, 0)
	if err != nil {
		logger.Error("GetLabelSeq fail.err:[%s]", err)
		return
	}
	curr_dt_stamp := time.Now().Unix()
	// log.Debug("GetLabelSeq:%d	%d	%d", labale, seq, curr_dt_stamp)

	uniq_id := fmt.Sprintf("%d%d", labale, seq)
	uniq_id2 := fmt.Sprintf("%d", curr_dt_stamp)

	dest, _ := UInt64Str2Base36(uniq_id)
	// log.Debug("UInt64Str2Base36 labale and seq:%s", dest)

	dest2, _ := UInt64Str2Base36(uniq_id2)
	// log.Debug("UInt64Str2Base36 timestamp:%s", dest2)

	transid = prefix + dest + dest2
	// log.Debug("UInt64Str2Base36:%s", transid)
	return
}

// RemoveStringSliceElement remove slice element
func RemoveStringSliceElement(source []string, element string) []string {
	var index = -1
	for i, v := range source {
		if v == element {
			index = i
			break
		}
	}
	if index == -1 {
		return source
	}
	var result []string
	result = append(result, source[:index]...)
	return append(result, source[index+1:]...)
}

func UInt64Str2Base36(src string) (dest string, err error) {
	if len(src) == 0 || len(src) > 19 {
		return "", errors.NewInternalError(errors.WithMsg("invalid src"))
	}

	llsrc, parse_err := strconv.ParseInt(src, 10, 64)
	if parse_err != nil {
		logger.Error("strconv.ParseInt fail. err = %v", parse_err)
		return "", errors.NewInternalError(errors.WithMsg("src = %v, ParseInt err = %v", src, parse_err))
	}

	dest = ConvertIteration(llsrc, 36)
	return dest, nil
}

func ConvertIteration(llsrc int64, base int) string {
	var result string
	var digit []byte = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	if base >= 2 && base <= 36 {
		for llsrc > 0 {
			remainder := int(llsrc % int64(base))
			result = string(append([]byte(result), digit[remainder]))
			llsrc /= int64(base)
		}
	}

	return reverseString(result)
}

func reverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

// GetLocalIP ....
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// Proto to JSON
func Pb2Json(m proto.Message) (string, error) {
	ma := jsonpb.Marshaler{
		EnumsAsInts:  false, // 枚举类型不转为int
		EmitDefaults: false, // 不展示默认值, 有默认值的字段
		Indent:       "",
		OrigName:     true, // 保留字段原始名称
		AnyResolver:  nil,
	}

	s, err := ma.MarshalToString(m)
	if err != nil {
		return "", errors.NewInternalError(errors.WithMsg("pb = %v proto to json error", m))
	}

	return s, nil
}

// JSON to Proto
func Json2Pb(s string, pb proto.Message) error {
	if err := jsonpb.UnmarshalString(s, pb); err != nil {
		return errors.NewInternalError(errors.WithMsg("content = %s json to pb error", s))
	}
	return nil
}

func CheckCurrentBetweenTime(start, end string) (bool, error) {
	if len(start) == 0 || len(end) == 0 {
		return false, fmt.Errorf("start or end is empty")
	}
	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", start, Loc)
	if err != nil {
		return false, fmt.Errorf("start: %v time format is error", start)
	}
	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", end, Loc)
	if err != nil {
		return false, fmt.Errorf("end: %v time format is error", end)
	}
	if endTime.Unix() <= startTime.Unix() {
		return false, fmt.Errorf("start: %v is smaller than end: %v", start, end)
	}
	currentTime := time.Now().Unix()
	if currentTime < endTime.Unix() && currentTime > startTime.Unix() {
		return true, nil
	}
	return false, nil
}

// CheckBetweenExcludeMax [start, end)
func CheckBetweenExcludeMax(start, end, target int) bool {
	if target >= start && target < end {
		return true
	}
	return false
}

// GenerateBetweenNum num between start and end
func GenerateBetweenNum(start, end int) int {
	return start + (rand.Intn(end - start + 1))
}

func GetTodayBeginTime() time.Time {
	currentTime := time.Now().In(Loc)
	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
	return startTime
}

func GetTodayEndTime() time.Time {
	currentTime := time.Now().In(Loc)
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, currentTime.Location())
	return endTime
}

func ReverseSlice(s interface{}) {
	size := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

// src formate : 2021-10-10
// delta : day number
// return formate : 2021-12-12
// 2021-10-10 -->  2021-12-12
func AddDate(src string, delta int) string {
	srcTime, _ := time.ParseInLocation("2006-01-02 15:04:05", src+" 00:00:00", util.Loc)
	dstdate := srcTime.AddDate(0, 0, delta).Format("2006-01-02") // 30天前的日期
	return dstdate
}
