//

package logger

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogFilePlugin_Type(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Should run successfully and return plugin type",
			want: pluginType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &LogFilePlugin{}
			got := p.Type()
			assert.Equal(t, got, tt.want)
		})
	}
}

// for log file decoder test
type fakeDecoder struct{}

func (decoder *fakeDecoder) Decode(_ interface{}) error {
	return errors.New("fake decoder")
}
