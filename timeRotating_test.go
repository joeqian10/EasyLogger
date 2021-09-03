package EasyLogger

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	ts := "2021-09-01"
	a, err := time.Parse(FileNameTimeFormat, ts)
	assert.Nil(t, err)
	ts2 := "2021-09-02_14:58:56"
	a2, err := time.Parse("2006-01-02_15:04:05", ts2)
	b := a2.Format(FileNameTimeFormat)
	nn, err := time.Parse(FileNameTimeFormat, b)
	assert.Nil(t, err)
	d := nn.Sub(a)
	assert.Equal(t, NanosecondPerDay, d)
}
