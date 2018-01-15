// Copyright (C) 2017, ccpaging <ccpaging@gmail.com>.  All rights reserved.

// Package logs provides level-based and extendable logging.
//
// The 3-layers
// 
// - Catalogs, defined as package global, module global, and new loggers.
//   Include: io.Witer, prefix, filter level, and filters  
// - Filters / appenders / outputs
//   Running as go routine.
//   Include: filter level, channel to transfer log messages, log writer
// - Filter Level and formatter, real output log writer
// 
// Enhanced Logging
//
// This is inspired by the logging functionality in log4go. Essentially, you create a Logger
// object and create output filters for it. You can send whatever you want to the Logger,
// and it will filter and formatter that based on your settings and send it to the outputs.
// This way, you can put as much debug code in your program as you want, and when you're done
// you can filter out the mundane messages so only the important ones show up.
//
// Utility functions are provided to make life easier. Here is some example code to get started:
//
// log := nxlog4go.nxlog4go(logs.DEBUG)
// log.AddFilter("log", nxlog4go.FINE, nxlog4go.NewFileLogWriter("example.log", 1))
// log.Info("The time is now: %s", time.LocalTime().Format("15:04:05 MST 2006/01/02"))
//
// Usage notes:
// - The utility functions (Info, Debug, Warn, etc) derive their source from the
//   calling function, and this incurs extra overhead. It can be disabled.
// - New field prefix is adding to LogRecorder to identify different module/package
//   in large project  
//
// Changes from log4go:
// - The external interface has remained mostly stable, but a lot of the
//   internals have been changed, so if you depended on any of this or created
//   your own LogWriter, then you will probably have to update your code.
// - In particular, Logger is now a structure include io.Writer and a filters map.

package nxlog4go

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"bytes"
	"time"
	"sync"
	"io"
)

// Version information
const (
	NXLOG4GO_VERSION = "nxlog4go-v0.0.1"
	NXLOG4GO_MAJOR   = 0
	NXLOG4GO_MINOR   = 0
	NXLOG4GO_BUILD   = 1
)

/****** Constants ******/

// These are the integer logging levels used by the logger
type Level int

const (
	FINEST Level = iota
	FINE
	DEBUG
	TRACE
	INFO
	WARNING
	ERROR
	CRITICAL
)

// Logging level strings
var (
	levelStrings = [...]string{"FNST", "FINE", "DEBG", "TRAC", "INFO", "WARN", "EROR", "CRIT"}
)

func (l Level) String() string {
	if l < 0 || int(l) > len(levelStrings) {
		return "UNKNOWN"
	}
	return levelStrings[int(l)]
}

/****** Variables ******/
var (
	// Default skip passed to runtime.Caller to get file name/line
	// May require tweaking if you want to wrap the logger
	DefaultCallerSkip = 2

	// Default buffer length specifies how many log messages a particular log4go
	// logger can buffer at a time before writing them.
	DefaultBufferLength = 32
)

/****** LogRecord ******/

// A LogRecord contains all of the pertinent information for each message
type LogRecord struct {
	Level   Level     // The log level
	Created time.Time // The time at which the log message was created (nanoseconds)
	Prefix  string    // The message prefix
	Source  string    // The message source
	Line	int 	  // The source line
	Message string    // The log message
}

/****** Logger ******/

// A Logger represents an active logging object that generates lines of
// output to an io.Writer, and a collection of Filters through which 
// log messages are written. Each logging operation makes a single call to
// the Writer's Write method. A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the Writer.
type Logger struct {
	mu     sync.Mutex // ensures atomic writes; protects the following fields
	out    io.Writer  // destination for output
	level  Level      // The log level
	caller bool  	  // runtime caller skip
	prefix string     // prefix to write at beginning of each line
	formatSlice [][]byte // Split the format into pieces by % signs
	filters map[string]*Filter // a collection of Filters
}

// New creates a new Logger. The out variable sets the
// destination to which log data will be written.
// The prefix appears at the beginning of each generated log line.
// The flag argument defines the logging properties.
func NewLogger(out io.Writer, lvl Level, prefix string, format string) *Logger {
	return &Logger{
		out: out,
		level: lvl,
		caller: true,
		prefix: prefix,
		formatSlice: bytes.Split([]byte(format), []byte{'%'}),
		filters: make(map[string]*Filter),
	}
}

// Create a new logger with a "stderr" writer to send log messages at
// or above lvl to standard output.
func New(lvl Level) *Logger {
	return NewLogger(os.Stderr, lvl, "", FORMAT_DEFAULT)
}

// Create a new logger copying from another logger 
func NewCopy(src *Logger, prefix string) *Logger {
	log := &Logger {
		out: src.out,
		level: src.level,
		caller: src.caller,
		prefix: src.prefix,
		filters: make(map[string]*Filter),
	
	}
	if len(prefix) > 0 {
		log.prefix = prefix
	}
	log.formatSlice = append(log.formatSlice, src.formatSlice...) 
	return log.CopyFilters(src)
}

// SetOutput sets the output destination for the logger.
func (log *Logger) SetOutput(w io.Writer) *Logger {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.out = w
	return log
}

// Level returns the output level for the logger.
func (log *Logger) Level() Level {
	log.mu.Lock()
	defer log.mu.Unlock()
	return log.level
}

// SetLevel sets the output level for the logger.
func (log *Logger) SetLevel(lvl Level) *Logger {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.level = lvl
	return log
}

// Return runtime caller skip for the logger.
func (log *Logger) Caller() bool {
	log.mu.Lock()
	defer log.mu.Unlock()
	return log.caller
}

// SetSkip sets the runtime caller skip for the logger.
// skip = -1, no runtime caller
func (log *Logger) SetCaller(caller bool) *Logger {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.caller = caller
	return log
}

// Format returns the output format for the logger.
func (log *Logger) Format() string {
	log.mu.Lock()
	defer log.mu.Unlock()
	return string(bytes.Join(log.formatSlice, []byte{'%'}))
}

// SetFormat sets the output flags for the logger.
func (log *Logger) SetFormat(format string) *Logger {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.formatSlice = bytes.Split([]byte(format), []byte{'%'})
	return log
}

// Prefix returns the output prefix for the logger.
func (log *Logger) Prefix() string {
	log.mu.Lock()
	defer log.mu.Unlock()
	return log.prefix
}

// SetPrefix sets the output prefix for the logger.
func (log *Logger) SetPrefix(prefix string) *Logger {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.prefix = prefix
	return log
}

// Add a new filter to the Logger which will only log messages at lvl or
// higher.  This function should not be called from multiple goroutines.
// Returns the logger for chaining.
func (log *Logger) AddFilter(name string, lvl Level, writer LogWriter) *Logger {
	log.filters[name] = NewFilter(lvl, writer)
	return log
}

// Copy all filters from another logger which is always Global logger. 
// Returns the logger for chaining.
func (log *Logger) CopyFilters(src *Logger) *Logger {
	log.RemoveFilters()
	for name, filt := range src.filters {
		log.AddFilter(name, filt.Level, filt.LogWriter)
	}
	return log
}

// Remove all filters writers in preparation for exiting the program or a
// reconfiguration of logging.  This DO NOT CLOSE filters.
// Returns the logger for chaining.
func (log *Logger) RemoveFilters() *Logger {
	// Close all filters
	for name, _ := range log.filters {
		delete(log.filters, name)
	}
	return log
}

// Close and remove all filters in preparation for exiting the program or a
// reconfiguration of logging.  Calling this is not really imperative, unless
// you want to guarantee that all log messages are written.  Close removes
// all filters (and thus all LogWriters) from the logger.
// Returns the logger for chaining.
func (log Logger) CloseFilters() *Logger {
	// Close all filters
	for _, filt := range log.filters {
		filt.Close()
	}
	return log.RemoveFilters()
}

/******* Logging *******/

// Determine if any logging will be done
func (log Logger) skip(lvl Level) bool {
	if log.out != nil && lvl >= log.level {
        return false
    }

	for _, filt := range log.filters {
		if lvl >= filt.Level {
			return false
		}
	}
	return true
}

func (log Logger) intMsg(arg0 interface{}, args ...interface{}) string {
	var msg string
	switch first := arg0.(type) {
	case string:
		// Use the string as a format string
		msg = fmt.Sprintf(first, args...)
	case func() string:
		// Log the closure (no other arguments used)
		msg = first()
	default:
		// Build a format string so that it will be similar to Sprint
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	return msg
}

// Send a log message with manual level, source, and message.
func (log Logger) intLog(lvl Level, arg0 interface{}, args ...interface{}) {
	log.mu.Lock()
	defer log.mu.Unlock()

	// Make the log record
	rec := &LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Prefix:  log.prefix,
		Message: log.intMsg(arg0, args...),
	}

	if log.caller {
		// Determine caller func - it's expensive.
		log.mu.Unlock()

		var ok bool
		_, rec.Source, rec.Line, ok = runtime.Caller(DefaultCallerSkip)
		if !ok {
			rec.Source = "???"
			rec.Line = 0
		}

		log.mu.Lock()
	}
	
	if log.out != nil {
        log.out.Write(FormatLogRecord(log.formatSlice, rec))
    }

	// Dispatch the logs
	for _, filt := range log.filters {
		if lvl < filt.Level {
			continue
		}
		filt.LogWrite(rec)
	}
}

// Finest logs a message at the finest log level.
// See Debug for an explanation of the arguments.
func (log Logger) Finest(arg0 interface{}, args ...interface{}) {
	if log.skip(FINEST) {
		return
	}
	log.intLog(FINEST, arg0, args...)
}

// Fine logs a message at the fine log level.
// See Debug for an explanation of the arguments.
func (log Logger) Fine(arg0 interface{}, args ...interface{}) {
	if log.skip(FINE) {
		return
	}
	log.intLog(FINE, arg0, args...)
}

// Debug is a utility method for debug log messages.
// The behavior of Debug depends on the first argument:
// - arg0 is a string
//   When given a string as the first argument, this behaves like Logf but with
//   the DEBUG log level: the first argument is interpreted as a format for the
//   latter arguments.
// - arg0 is a func()string
//   When given a closure of type func()string, this logs the string returned by
//   the closure iff it will be logged.  The closure runs at most one time.
// - arg0 is interface{}
//   When given anything else, the log message will be each of the arguments
//   formatted with %v and separated by spaces (ala Sprint).
func (log Logger) Debug(arg0 interface{}, args ...interface{}) {
	if log.skip(DEBUG) {
		return
	}
	log.intLog(DEBUG, arg0, args...)
}

// Trace logs a message at the trace log level.
// See Debug for an explanation of the arguments.
func (log Logger) Trace(arg0 interface{}, args ...interface{}) {
	if log.skip(TRACE) {
		return
	}
	log.intLog(TRACE, arg0, args...)
}

// Info logs a message at the info log level.
// See Debug for an explanation of the arguments.
func (log Logger) Info(arg0 interface{}, args ...interface{}) {
	if log.skip(INFO) {
		return
	}
	log.intLog(INFO, arg0, args...)
}

// Warn logs a message at the warning log level and returns the formatted error.
// At the warning level and higher, there is no performance benefit if the
// message is not actually logged, because all formats are processed and all
// closures are executed to format the error message.
// See Debug for further explanation of the arguments.
func (log Logger) Warn(arg0 interface{}, args ...interface{}) error {
	msg := log.intMsg(arg0, args...)
	log.intLog(WARNING, msg)
	return errors.New(msg)
}

// Error logs a message at the error log level and returns the formatted error,
// See Warn for an explanation of the performance and Debug for an explanation
// of the parameters.
func (log Logger) Error(arg0 interface{}, args ...interface{}) error {
	msg := log.intMsg(arg0, args...)
	log.intLog(ERROR, msg)
	return errors.New(msg)
}

// Critical logs a message at the critical log level and returns the formatted error,
// See Warn for an explanation of the performance and Debug for an explanation
// of the parameters.
func (log Logger) Critical(arg0 interface{}, args ...interface{}) error {
	msg := log.intMsg(arg0, args...)
	log.intLog(CRITICAL, msg)
	return errors.New(msg)
}