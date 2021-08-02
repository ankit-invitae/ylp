package util

import (
	"fmt"
	"os"
)

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func SaveTofile(file *os.File, data string) {
	file.WriteString(data)
	defer file.Close()
}
