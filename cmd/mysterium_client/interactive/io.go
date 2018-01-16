package interactive

import (
	"fmt"
)

const statusColor = "\033[33m"
const warningColor = "\033[31m"
const successColor = "\033[32m"
const infoColor = "\033[93m"

func status(label string, items ...interface{}) {
	fmt.Printf(statusColor+"[%s] \033[0m", label)
	fmt.Println(items...)
}

func warn(items ...interface{}) {
	fmt.Printf(warningColor + "[WARNING] \033[0m")
	fmt.Println(items...)
}

func success(items ...interface{}) {
	fmt.Printf(successColor + "[SUCCESS] \033[0m")
	fmt.Println(items...)
}

func info(items ...interface{}) {
	fmt.Printf(infoColor + "[INFO] \033[0m")
	fmt.Println(items...)
}
