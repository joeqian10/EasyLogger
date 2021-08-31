package EasyLogger

import (
	"log"
	"testing"
)

func TestNewRotatingEasyLogger(t *testing.T) {
	l := NewRotatingEasyLogger("./Logs/test.log",
		1,
		30,
		30,
		true,
		false,
		log.Ldate | log.Lmicroseconds,
		"",
		true)

	for i:= 0; i<10000;i++ {
		l.Trace("hello world")
		l.Debug("hello world")
		l.Info("hello world")
		l.Warn("hello world")
		l.Error("hello world")
		l.Fatal("hello world")

		l.Tracef("f:%s", "hello world")
		l.Debugf("f:%s", "hello world")
		l.Infof("f:%s", "hello world")
		l.Warnf("f:%s", "hello world")
		l.Errorf("f:%s", "hello world")
		l.Fatalf("f:%s", "hello world")
	}
}
