package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

var logfile *os.File
func loggerInit() {
	// logfile = nil
	if logpath != "" {
		loggerRotate()
	}
}
func loggerStop() {
	if logfile != nil {
		logfile.Close()
	}
}

func loggerRotate() {
	if logpath == "" {
		return
	}
	fmt.Printf("%s [INFO] 重载日志文件.\n", time.Now().Local().Format("2006-01-02 15:04:05"))
	logfile2, err := os.OpenFile(logpath, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0777)
	if err != nil {
		log.Fatal("[FATAL] 无法打开日志文件进行输出:", err)
	}
	log.SetOutput(logfile2)
	if logfile != nil {
		logfile.Close()
	}
	logfile = logfile2
}
