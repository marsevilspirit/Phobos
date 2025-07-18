package log

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

type defaultLogger struct {
	*log.Logger
}

func (l *defaultLogger) Debug(v ...any) {
	l.Output(calldepth, header("DEBUG", fmt.Sprint(v...)))
}
func (l *defaultLogger) Debugf(format string, v ...any) {
	l.Output(calldepth, header("DEBUG", fmt.Sprintf(format, v...)))
}

func (l *defaultLogger) Info(v ...any) {
	l.Output(calldepth, header(color.GreenString("INFO "), fmt.Sprint(v...)))
}
func (l *defaultLogger) Infof(format string, v ...any) {
	l.Output(calldepth, header(color.GreenString("INFO "), fmt.Sprintf(format, v...)))
}

func (l *defaultLogger) Warn(v ...any) {
	l.Output(calldepth, header(color.YellowString("WARN "), fmt.Sprint(v...)))
}
func (l *defaultLogger) Warnf(format string, v ...any) {
	l.Output(calldepth, header(color.YellowString("WARN "), fmt.Sprintf(format, v...)))
}

func (l *defaultLogger) Error(v ...any) {
	l.Output(calldepth, header(color.RedString("ERROR"), fmt.Sprint(v...)))
}
func (l *defaultLogger) Errorf(format string, v ...any) {
	l.Output(calldepth, header(color.RedString("ERROR"), fmt.Sprintf(format, v...)))
}

func (l *defaultLogger) Fatal(v ...any) {
	l.Output(calldepth, header(color.MagentaString("FATAL"), fmt.Sprint(v...)))
	os.Exit(1)
}
func (l *defaultLogger) Fatalf(format string, v ...any) {
	l.Output(calldepth, header(color.MagentaString("FATAL"), fmt.Sprintf(format, v...)))
	os.Exit(1)
}

func (l *defaultLogger) Panic(v ...any) {
	l.Logger.Panic(v...)
}
func (l *defaultLogger) Panicf(format string, v ...any) {
	l.Logger.Panicf(format, v...)
}

func header(lvl, msg string) string {
	return fmt.Sprintf("%s: %s", lvl, msg)
}
