package wa

import (
	"os"
	"strings"

	qrterminal "github.com/mdp/qrterminal/v3"
)

func printTerminalQR(code string) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return
	}
	config := qrterminal.Config{
		Level:     qrterminal.M,
		Writer:    os.Stdout,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 1,
	}
	qrterminal.GenerateWithConfig(trimmed, config)
}
