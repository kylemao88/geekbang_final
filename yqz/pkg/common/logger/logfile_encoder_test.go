//

package logger

import (
	"encoding/json"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func Test_logfileEncoder_tryAddRuneSelf(t *testing.T) {
	type fields struct {
		EncoderConfig  *zapcore.EncoderConfig
		buf            *buffer.Buffer
		spaced         bool
		openNamespaces int
		skipStep       int
		reflectBuf     *buffer.Buffer
		reflectEnc     *json.Encoder
	}
	type args struct {
		b byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "Should return false for large than one byte",
			fields: fields{},
			args:   args{b: utf8.RuneSelf + 1},
			want:   false,
		},
		{
			name:   "Should return true for branch double \\",
			fields: fields{buf: &buffer.Buffer{}},
			args:   args{b: '\\'},
			want:   true,
		},
		{
			name:   "Should return true for branch \"",
			fields: fields{buf: &buffer.Buffer{}},
			args:   args{b: '"'},
			want:   true,
		},
		{
			name:   "Should return true for branch \\n",
			fields: fields{buf: &buffer.Buffer{}},
			args:   args{b: '\n'},
			want:   true,
		},
		{
			name:   "Should return true for branch \\r",
			fields: fields{buf: &buffer.Buffer{}},
			args:   args{b: '\r'},
			want:   true,
		},
		{
			name:   "Should return true for branch \\t",
			fields: fields{buf: &buffer.Buffer{}},
			args:   args{b: '\t'},
			want:   true,
		},
		{
			name:   "Should return true for branch >=0x20",
			fields: fields{buf: &buffer.Buffer{}},
			args:   args{b: 0x20 + 1},
			want:   true,
		},
		{
			name:   "Should return true for branch <0x20",
			fields: fields{buf: &buffer.Buffer{}},
			args:   args{b: 0x20 - 1},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := &logfileEncoder{
				EncoderConfig: tt.fields.EncoderConfig,
				buf:           tt.fields.buf,
				skipStep:      tt.fields.skipStep,

				reflectBuf: tt.fields.reflectBuf,
				reflectEnc: tt.fields.reflectEnc,
			}
			got := encoder.tryAddRuneSelf(tt.args.b)
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_format(t *testing.T) {
	type args struct {
		pattern string
		key     string
		value   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Should return [key1:value1]",
			args: args{
				pattern: "[%k:%v]",
				key:     "key1",
				value:   "value1",
			},
			want: "[key1:value1]",
		},
		{
			name: "Should return key1=value1",
			args: args{
				pattern: "%k=%v",
				key:     "key1",
				value:   "value1",
			},
			want: "key1=value1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := format(tt.args.pattern, tt.args.key, tt.args.value)
			assert.Equal(t, got, tt.want)
		})
	}
}
