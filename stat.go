package main

import (
	"log"
	"strconv"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

func cpuStatCsvHeader() [10]string {
	return [10]string{
		"id",
		"system",
		"user",
		"idle",
		"nice",
		"ioWait",
		"irq",
		"softIrq",
		"steal",
		"guest",
	}
}

func cpuStatCsv() [10]string {
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		log.Fatal("could not read /proc/stat")
	}
	a := stat.CPUStatAll
	return [10]string{
		a.Id,
		strconv.FormatInt(a.System),
		string(a.User),
		string(a.Idle),
		string(a.Nice),
		string(a.IOWait),
		string(a.IRQ),
		string(a.SoftIRQ),
		string(a.Steal),
		string(a.Guest),
	}
}
