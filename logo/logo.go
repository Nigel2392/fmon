package logo

import (
	"fmt"
	"os"
	"strings"

	"github.com/Nigel2392/fmon/ttext"
)

var reset = "\033[0m"
var colorl = []string{
	"\033[31m",
	"\033[31m",
	"\033[35m",
	"\033[34m",
	"\033[34m",
	"\033[36m",
	"\033[32m",
	"\033[32m",
	"\033[33m",

	"\033[37m",
	"\033[97m",
}

var NO_COLOR = os.Getenv("NO_COLOR") != "" ||
	os.Getenv("TERM") == "dumb" ||
	os.Getenv("COLORTERM") == "false"

var FMON_LOGO = ttext.Sentence("FMon", func(maxL int, curL int, s string) string {
	if NO_COLOR {
		return s
	}
	s = fmt.Sprintf("%s%s%s", colorl[curL%len(colorl)], s, reset)
	return s
})

func Print() {
	var sub = 9
	if NO_COLOR {
		sub = 0
	}

	println(
		FMON_LOGO,
		fmt.Sprintf(
			"%s",
			strings.Repeat("-", len(strings.Split(FMON_LOGO, "\n")[0])-sub), // 10 to account for ansi escape
		),
	)
}
