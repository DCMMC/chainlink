package logger

import (
	"go.uber.org/zap/zapcore"
)

var NullLogger Logger

func init() {
	NullLogger = NewNullLogger()
}

type nullLogger struct{}

func NewNullLogger() Logger {
	return &nullLogger{}
}

func (l *nullLogger) With(args ...interface{}) Logger                 { return l }
func (l *nullLogger) Named(name string) Logger                        { return l }
func (l *nullLogger) NewRootLogger(lvl zapcore.Level) (Logger, error) { return l, nil }
func (l *nullLogger) SetLogLevel(_ zapcore.Level)                     {}
func (l *nullLogger) Debug(args ...interface{})                       {}
func (l *nullLogger) Info(args ...interface{})                        {}
func (l *nullLogger) Warn(args ...interface{})                        {}
func (l *nullLogger) Error(args ...interface{})                       {}
func (l *nullLogger) Fatal(args ...interface{})                       {}
func (l *nullLogger) Panic(args ...interface{})                       {}
func (l *nullLogger) Debugf(format string, values ...interface{})     {}
func (l *nullLogger) Infof(format string, values ...interface{})      {}
func (l *nullLogger) Warnf(format string, values ...interface{})      {}
func (l *nullLogger) Errorf(format string, values ...interface{})     {}
func (l *nullLogger) Fatalf(format string, values ...interface{})     {}
func (l *nullLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (l *nullLogger) Infow(msg string, keysAndValues ...interface{})  {}
func (l *nullLogger) Warnw(msg string, keysAndValues ...interface{})  {}
func (l *nullLogger) Errorw(msg string, keysAndValues ...interface{}) {}
func (l *nullLogger) Fatalw(msg string, keysAndValues ...interface{}) {}
func (l *nullLogger) Panicw(msg string, keysAndValues ...interface{}) {}
func (l *nullLogger) WarnIf(err error, msg string)                    {}
func (l *nullLogger) ErrorIf(err error, msg string)                   {}
func (l *nullLogger) PanicIf(err error, msg string)                   {}
func (l *nullLogger) ErrorIfCalling(fn func() error)                  {}
func (l *nullLogger) Sync() error                                     { return nil }
func (l *nullLogger) withCallerSkip(add int) Logger                   { return l }
