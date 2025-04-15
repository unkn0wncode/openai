package util

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
)

// LogStack prints whole stack starting from where it was called to a given log
func LogStack(logger *log.Logger) {
	level := 1
	for {
		// fmt.Println("LogStack level", level)
		pc, _, _, ok := runtime.Caller(level)
		if !ok {
			// fmt.Println("Break: no pc")
			break
		}
		callerFunc := runtime.FuncForPC(pc)
		if callerFunc == nil {
			// fmt.Println("Break: nill caller")
			break
		}
		caller := callerFunc.Name()
		logger.Output(level+1, caller)
		level++
	}
}

// Dump returns a full dump of http request/response.
func Dump[T *http.Request | *http.Response](r T) ([]byte, error) {
	switch r := any(r).(type) {
	case *http.Request:
		if r.RequestURI != "" {
			return httputil.DumpRequest(r, true)
		}
		return httputil.DumpRequestOut(r, true)
	case *http.Response:
		return httputil.DumpResponse(r, true)
	default:
		return nil, fmt.Errorf("unsupported type %T", r)
	}
}

// CompactStack returns a compact stack trace looking like:
// "package.function:line > package.function:line > ...".
func CompactStack() string {
	frameMatcher := regexp.MustCompile(`(?m)^([^\n\s]+\.\w+)[^\n]*\n[^\n]*(:\d+)`)
	stack := string(debug.Stack())
	frames := frameMatcher.FindAllStringSubmatch(stack, -1)
	slices.Reverse(frames)

	var frameTexts []string
	for _, frame := range frames {
		if strings.Contains(frame[0], "debug.Stack") {
			continue
		}
		frameTexts = append(frameTexts, fmt.Sprintf("%s%s", frame[1], frame[2]))
	}

	return strings.Join(frameTexts, " > ")
}
