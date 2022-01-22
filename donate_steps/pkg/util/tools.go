//

package util

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
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
func GetUsedTime(msg string) func() time.Duration {
	start := time.Now()
	return func() time.Duration {
		return time.Since(start)
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

// FormatTime format time
func FormatTime(timestamp int64) string {
	sec := time.Duration(timestamp).Seconds()
	tm := time.Unix(int64(sec), 0)
	return tm.Format("2006-01-02 15:04:05")
}

// DeepCopy is used to copy struct
func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
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

// GetFirstDateOfWeek return first day date of week
func GetFirstDateOfWeek() (weekMonday string) {
	now := GetLocalTime()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	weekStartDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekMonday = weekStartDate.Format("2006-01-02")
	return
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

// GenerateBetweenNum num between start and end
func GenerateBetweenNum(start, end int) int {
	return start + (rand.Intn(end - start + 1))
}
