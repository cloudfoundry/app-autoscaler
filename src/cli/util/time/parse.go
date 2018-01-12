package time

import (
	"cli/ui"
	"errors"
	"fmt"
	"time"
)

var timeFormats = []string{
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05-07:00",
}

func ParseTimeFormat(input string) (ns int64, e error) {

	if input != "" {
		for _, format := range timeFormats {
			t, e := time.Parse(format, input)
			if e == nil {
				ns = t.UnixNano()
				return ns, nil
			}
		}
	}
	e = errors.New(fmt.Sprintf(ui.UnrecognizedTimeFormat, input))
	return
}
