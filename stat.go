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
		strconv.FormatUint(a.System, 10),
		strconv.FormatUint(a.User, 10),
		strconv.FormatUint(a.Idle, 10),
		strconv.FormatUint(a.Nice, 10),
		strconv.FormatUint(a.IOWait, 10),
		strconv.FormatUint(a.IRQ, 10),
		strconv.FormatUint(a.SoftIRQ, 10),
		strconv.FormatUint(a.Steal, 10),
		strconv.FormatUint(a.Guest, 10),
	}
}
