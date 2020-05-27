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

func cpuStatToStr(stat uint64) string {
	return strconv.FormatUint(stat, 10)
}

func cpuStatCsv() [10]string {
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		log.Fatal("could not read /proc/stat")
	}
	a := stat.CPUStatAll
	return [10]string{
		a.Id,
		cpuStatToStr(a.System),
		cpuStatToStr(a.User),
		cpuStatToStr(a.Idle),
		cpuStatToStr(a.Nice),
		cpuStatToStr(a.IOWait),
		cpuStatToStr(a.IRQ),
		cpuStatToStr(a.SoftIRQ),
		cpuStatToStr(a.Steal),
		cpuStatToStr(a.Guest),
	}
}
