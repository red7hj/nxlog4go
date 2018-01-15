// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package filelog

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"

	l4g "github.com/ccpaging/nxlog4go"
)

const testLogFile = "_logtest.log"
const benchLogFile = "_benchlog.log"

var now time.Time = time.Unix(0, 1234567890123456789).In(time.UTC)

func newLogRecord(lvl l4g.Level, src string, msg string) *l4g.LogRecord {
	return &l4g.LogRecord{
		Level:   lvl,
		Source:  src,
		Created: now,
		Message: msg,
	}
}

func TestFileLogWriter(t *testing.T) {
	w := NewFileLogWriter(testLogFile, 0)
	if w == nil {
		t.Fatalf("Invalid return: w should not be nil")
	}
	defer os.Remove(testLogFile)

	w.LogWrite(newLogRecord(l4g.CRITICAL, "source", "message"))
	runtime.Gosched()
	w.Close()

	if contents, err := ioutil.ReadFile(testLogFile); err != nil {
		t.Errorf("read(%q): %s", testLogFile, err)
	} else if len(contents) != 50 {
		t.Errorf("malformed filelog: %q (%d bytes)", string(contents), len(contents))
	}
}

func BenchmarkFileLog(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil).SetCaller(false)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 0))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Warn("This is a log message")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

func BenchmarkFileNotLogged(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil).SetCaller(false)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 0))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Debug("This is a log message")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

func BenchmarkFileUtilLog(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 0))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Info("%s is a log message", "This")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

func BenchmarkFileUtilNotLog(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 0))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Debug("%s is a log message", "This")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

func BenchmarkCacheFileLog(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil).SetCaller(false)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 4096))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Warn("This is a log message")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

func BenchmarkCacheFileNotLogged(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil).SetCaller(false)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 4096))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Debug("This is a log message")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

func BenchmarkCacheFileUtilLog(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 4096))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Info("%s is a log message", "This")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

func BenchmarkCacheFileUtilNotLog(b *testing.B) {
	sl := l4g.New(l4g.INFO).SetOutput(nil)
	b.StopTimer()
	sl.AddFilter("file", l4g.INFO, NewFileLogWriter(benchLogFile, 0).Set("flush", 4096))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sl.Debug("%s is a log message", "This")
	}
	b.StopTimer()
	sl.CloseFilters()
	os.Remove(benchLogFile)
}

// Benchmark results (darwin amd64 6g)
//elog.BenchmarkConsoleLog           100000       22819 ns/op
//elog.BenchmarkConsoleNotLogged    2000000         879 ns/op
//elog.BenchmarkConsoleUtilLog        50000       34380 ns/op
//elog.BenchmarkConsoleUtilNotLog   1000000        1339 ns/op
//elog.BenchmarkFileLog              100000       26497 ns/op
//elog.BenchmarkFileNotLogged       2000000         821 ns/op
//elog.BenchmarkFileUtilLog           50000       33945 ns/op
//elog.BenchmarkFileUtilNotLog      1000000        1258 ns/op