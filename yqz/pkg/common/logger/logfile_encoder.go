//

package logger

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// The key and value placeholder in pattern.
const (
	// key value format also used by logfile_writer
	KeyFormat    = "%k"
	ValueFormat  = "%v"
	headSplitter = '|'
)

// For JSON-escaping; see logfileEncoder.safeAddString below.
const _hex = "0123456789abcdef"

var bufferPool = buffer.NewPool()

var _logfilePool = sync.Pool{New: func() interface{} {
	return &logfileEncoder{}
}}

func getLogFileEncoder() *logfileEncoder {
	return _logfilePool.Get().(*logfileEncoder)
}

func putLogFileEncoder(encoder *logfileEncoder) {
	if encoder.reflectBuf != nil {
		encoder.reflectBuf.Free()
	}
	encoder.EncoderConfig = nil
	encoder.goIDKey = ""
	encoder.buf = nil
	encoder.pattern = ""
	encoder.fieldsSplitter = ""
	encoder.reflectBuf = nil
	encoder.reflectEnc = nil
	_logfilePool.Put(encoder)
}

// logfileEncoder encode entry according to customized format.
type logfileEncoder struct {
	*zapcore.EncoderConfig
	goIDKey  string
	buf      *buffer.Buffer
	skipStep int // use to get function name

	// for formatting entry
	pattern        string
	fieldsSplitter string

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc *json.Encoder
}

// NewLogFileEncoder creates a logfileEncoder by config
func NewLogFileEncoder(encoderConfig zapcore.EncoderConfig, skipStep int,
	pattern, fieldsSplitter string, goIDKey string) zapcore.Encoder {
	return newLogFileEncoder(encoderConfig, skipStep, pattern, fieldsSplitter, goIDKey)
}

func newLogFileEncoder(encoderConfig zapcore.EncoderConfig, skipStep int,
	pattern, fieldsSplitter string, goIDKey string) *logfileEncoder {
	return &logfileEncoder{
		EncoderConfig:  &encoderConfig,
		goIDKey:        goIDKey,
		buf:            bufferPool.Get(),
		skipStep:       skipStep,
		pattern:        pattern,
		fieldsSplitter: fieldsSplitter,
	}
}

// AddArray is interface required function that not used.
func (encoder *logfileEncoder) AddArray(_ string, _ zapcore.ArrayMarshaler) error {
	return nil
}

// AddObject is interface required function that not used.
func (encoder *logfileEncoder) AddObject(_ string, _ zapcore.ObjectMarshaler) error {
	return nil
}

// AddBinary is interface required function that not used.
func (encoder *logfileEncoder) AddBinary(_ string, _ []byte) {
	return
}

// AddByteString is interface required function that not used.
func (encoder *logfileEncoder) AddByteString(_ string, _ []byte) {
	return
}

// AddBool is interface required function that not used.
func (encoder *logfileEncoder) AddBool(_ string, _ bool) {
	return
}

// AddComplex128 is interface required function that not used.
func (encoder *logfileEncoder) AddComplex128(_ string, _ complex128) {
	return
}

// AddDuration is interface required function that not used.
func (encoder *logfileEncoder) AddDuration(_ string, _ time.Duration) {
	return
}

// AddFloat64 is interface required function that not used.
func (encoder *logfileEncoder) AddFloat64(_ string, _ float64) {
	return
}

// AddInt64 is interface required function that not used.
func (encoder *logfileEncoder) AddInt64(_ string, _ int64) {
	return
}

// resetReflectBuf is interface required function that not used.
func (encoder *logfileEncoder) resetReflectBuf() {
	return
}

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (encoder *logfileEncoder) encodeReflected(_ interface{}) ([]byte, error) {
	return nil, nil
}

// AddReflected is interface required function that not used.
func (encoder *logfileEncoder) AddReflected(_ string, _ interface{}) error {
	return nil
}

// OpenNamespace is interface required function that not used.
func (encoder *logfileEncoder) OpenNamespace(_ string) {
	return
}

// AddString adds string in pattern defined way to encoder's buffer.
func (encoder *logfileEncoder) AddString(key, val string) {
	encoder.addElementSeparator()
	encoder.buf.AppendString(format(encoder.pattern, key, val))
}

// Format formats key-value pair in defined pattern.
func format(pattern, key, value string) string {
	result := ""
	tmp := strings.Replace(pattern, KeyFormat, key, 1)
	result += strings.Replace(tmp, ValueFormat, value, 1)

	return result
}

// AddTime is interface required function that not used.
func (encoder *logfileEncoder) AddTime(_ string, _ time.Time) {
	return
}

// AddUint64 is interface required function that not used.
func (encoder *logfileEncoder) AddUint64(_ string, _ uint64) {
	return
}

// AppendArray is interface required function that not used.
func (encoder *logfileEncoder) AppendArray(_ zapcore.ArrayMarshaler) error {
	return nil
}

// AppendObject is interface required function that not used.
func (encoder *logfileEncoder) AppendObject(_ zapcore.ObjectMarshaler) error {
	return nil
}

// AppendBool is interface required function that not used.
func (encoder *logfileEncoder) AppendBool(_ bool) {
	return
}

// AppendByteString appends byte string value to encoder's buffer.
func (encoder *logfileEncoder) AppendByteString(val []byte) {
	encoder.safeAddByteString(val)
}

// AppendComplex128 is interface required function that not used.
func (encoder *logfileEncoder) AppendComplex128(_ complex128) {
	return
}

// AppendDuration is interface required function that not used.
func (encoder *logfileEncoder) AppendDuration(_ time.Duration) {
	return
}

// AppendInt64 is interface required function that not used.
func (encoder *logfileEncoder) AppendInt64(_ int64) {
	return
}

// AppendReflected is interface required function that not used.
func (encoder *logfileEncoder) AppendReflected(_ interface{}) error {
	return nil
}

// AppendString appends string value to encoder's buffer.
func (encoder *logfileEncoder) AppendString(val string) {
	encoder.safeAddString(val)
}

// AppendTimeLayout is interface required function that not used.
func (encoder *logfileEncoder) AppendTimeLayout(_ time.Time, _ string) {
	return
}

// AppendTime appends time.Time value to encoder's buffer.
func (encoder *logfileEncoder) AppendTime(val time.Time) {
	cur := encoder.buf.Len()
	encoder.EncodeTime(val, encoder)
	if cur == encoder.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		encoder.AppendInt64(val.UnixNano())
	}
}

// AppendUint64 is interface required function that not used.
func (encoder *logfileEncoder) AppendUint64(_ uint64) {
	return
}

// AddComplex64 is interface required function that not used.
func (encoder *logfileEncoder) AddComplex64(k string, v complex64) {
	encoder.AddComplex128(k, complex128(v))
}

// AddFloat32 ...
func (encoder *logfileEncoder) AddFloat32(k string, v float32) { encoder.AddFloat64(k, float64(v)) }

// AddInt ...
func (encoder *logfileEncoder) AddInt(k string, v int) { encoder.AddInt64(k, int64(v)) }

// AddInt32 ...
func (encoder *logfileEncoder) AddInt32(k string, v int32) { encoder.AddInt64(k, int64(v)) }

// AddInt16 ...
func (encoder *logfileEncoder) AddInt16(k string, v int16) { encoder.AddInt64(k, int64(v)) }

// AddInt8 ...
func (encoder *logfileEncoder) AddInt8(k string, v int8) { encoder.AddInt64(k, int64(v)) }

// AddUint ...
func (encoder *logfileEncoder) AddUint(k string, v uint) { encoder.AddUint64(k, uint64(v)) }

// AddUint32 ...
func (encoder *logfileEncoder) AddUint32(k string, v uint32) { encoder.AddUint64(k, uint64(v)) }

// AddUint16 ...
func (encoder *logfileEncoder) AddUint16(k string, v uint16) { encoder.AddUint64(k, uint64(v)) }

// AddUint8 ...
func (encoder *logfileEncoder) AddUint8(k string, v uint8) { encoder.AddUint64(k, uint64(v)) }

// AddUintptr ...
func (encoder *logfileEncoder) AddUintptr(k string, v uintptr) { encoder.AddUint64(k, uint64(v)) }

// AppendComplex64 ...
func (encoder *logfileEncoder) AppendComplex64(v complex64) { encoder.AppendComplex128(complex128(v)) }

// AppendFloat64 ...
func (encoder *logfileEncoder) AppendFloat64(v float64) { encoder.appendFloat(v, 64) }

// AppendFloat32 ...
func (encoder *logfileEncoder) AppendFloat32(v float32) { encoder.appendFloat(float64(v), 32) }

// AppendInt ...
func (encoder *logfileEncoder) AppendInt(v int) { encoder.AppendInt64(int64(v)) }

// AppendInt32 ...
func (encoder *logfileEncoder) AppendInt32(v int32) { encoder.AppendInt64(int64(v)) }

// AppendInt16 ...
func (encoder *logfileEncoder) AppendInt16(v int16) { encoder.AppendInt64(int64(v)) }

// AppendInt8 ...
func (encoder *logfileEncoder) AppendInt8(v int8) { encoder.AppendInt64(int64(v)) }

// AppendUint ...
func (encoder *logfileEncoder) AppendUint(v uint) { encoder.AppendUint64(uint64(v)) }

// AppendUint32 ...
func (encoder *logfileEncoder) AppendUint32(v uint32) { encoder.AppendUint64(uint64(v)) }

// AppendUint16 ...
func (encoder *logfileEncoder) AppendUint16(v uint16) { encoder.AppendUint64(uint64(v)) }

// AppendUint8 ...
func (encoder *logfileEncoder) AppendUint8(v uint8) { encoder.AppendUint64(uint64(v)) }

// AppendUintptr ...
func (encoder *logfileEncoder) AppendUintptr(v uintptr) { encoder.AppendUint64(uint64(v)) }

// Clone deeply clones a logfileEncoder
func (encoder *logfileEncoder) Clone() zapcore.Encoder {
	clone := encoder.clone()
	clone.buf.Write(encoder.buf.Bytes())
	return clone
}

func (encoder *logfileEncoder) clone() *logfileEncoder {
	clone := getLogFileEncoder()
	clone.EncoderConfig = encoder.EncoderConfig
	clone.goIDKey = encoder.goIDKey
	clone.buf = bufferPool.Get()
	clone.skipStep = encoder.skipStep
	clone.pattern = encoder.pattern
	clone.fieldsSplitter = encoder.fieldsSplitter
	return clone
}

// EncodeEntry encodes Entry according to wanted format.
func (encoder *logfileEncoder) EncodeEntry(ent zapcore.Entry, _ []zapcore.Field) (*buffer.Buffer, error) {
	final := encoder.clone()

	// time
	if final.TimeKey != "" {
		final.AppendTime(ent.Time)
		final.buf.AppendByte(headSplitter)
	}

	/*
		// clusterName
		final.buf.AppendString(os.Getenv("MY_CLUSTER"))
		final.buf.AppendByte(headSplitter)

		// nodeName
		final.buf.AppendString(os.Getenv("MY_CLUSTER"))
		final.buf.AppendByte(headSplitter)

		// podName
		final.buf.AppendString(os.Getenv("MY_POD"))
		final.buf.AppendByte(headSplitter)
	*/

	// log level
	if final.LevelKey != "" {
		msg := fmt.Sprintf("%6v", ent.Level.CapitalString())
		final.buf.AppendString(msg)
		final.buf.AppendByte(headSplitter)
	}

	/*
		// goroutine id
		if final.goIDKey != "" {
			msg := fmt.Sprintf("%6v", getGoID())
			final.buf.AppendString(msg)
			final.buf.AppendByte(headSplitter)
		}
	*/

	// file:num:function
	if ent.Caller.Defined && final.CallerKey != "" {
		// final.EncodeCaller(ent.Caller, final)
		msg := fmt.Sprintf("%-56v ", ent.Caller.TrimmedPath()+":"+getFuncName(ent.Caller.PC))
		//msg := fmt.Sprintf("%-100v ", ent.Caller.FullPath()+":"+getFuncName(ent.Caller.PC))
		final.buf.AppendString(msg)
		final.buf.AppendByte(headSplitter)
	}

	if final.MessageKey != "" && ent.Message != "" {
		final.buf.AppendString(ent.Message)
	}

	if encoder.buf.Len() > 0 {
		final.buf.Write(encoder.buf.Bytes())
	}

	if ent.Stack != "" && final.StacktraceKey != "" {
		final.AddString(final.StacktraceKey, ent.Stack)
	}

	if final.LineEnding != "" {
		final.buf.AppendString(final.LineEnding)
	} else {
		final.buf.AppendString(zapcore.DefaultLineEnding)
	}

	ret := final.buf
	putLogFileEncoder(final)
	return ret, nil
}

func (encoder *logfileEncoder) closeOpenNamespaces() {
	return
}

func (encoder *logfileEncoder) addKey(_ string) {
	return
}

func (encoder *logfileEncoder) addElementSeparator() {
	last := encoder.buf.Len() - 1
	if last < 0 {
		return
	}
	encoder.buf.AppendString(encoder.fieldsSplitter)
}

func (encoder *logfileEncoder) appendFloat(_ float64, _ int) {
	return
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (encoder *logfileEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if encoder.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if encoder.tryAddRuneError(r, size) {
			i++
			continue
		}
		encoder.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (encoder *logfileEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if encoder.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if encoder.tryAddRuneError(r, size) {
			i++
			continue
		}
		encoder.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (encoder *logfileEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		encoder.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		encoder.buf.AppendByte('\\')
		encoder.buf.AppendByte(b)
	case '\n':
		encoder.buf.AppendByte('\\')
		encoder.buf.AppendByte('n')
	case '\r':
		encoder.buf.AppendByte('\\')
		encoder.buf.AppendByte('r')
	case '\t':
		encoder.buf.AppendByte('\\')
		encoder.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		encoder.buf.AppendString(`\u00`)
		encoder.buf.AppendByte(_hex[b>>4])
		encoder.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (encoder *logfileEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		encoder.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func getFuncName(pc uintptr) string {
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	// lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash
	firstDot := strings.IndexByte(funcName[lastSlash:], '.') + lastSlash
	if firstDot < 0 {
		firstDot = 0
	}
	// return funcName[lastSlash+1:]
	return funcName[firstDot+1:]
}

// getGoID get goroutine id
func getGoID() int64 {
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
