package main

import (
	"log"
	"strconv"

	procstat "github.com/shirou/gopsutil/cpu"
)

func cpuStatCsvHeader() []string {
	return []string{
		"time.total",
		"time.system",
		"time.user",
		"time.idle",
		"time.nice",
		"time.ioWait",
		"time.irq",
		"time.softIrq",
		"time.steal",
		"time.guest",
		"perc.total",
	}
}

func cpuStatToStr(stat float64) string {
	return strconv.FormatFloat(stat, 'f', 10, 64)
}

func cpuStatCsv() []string {
	tstat, err := procstat.Times(false)
	if err != nil {
		log.Fatal("could not read /proc/stat")
	}
	pstat, err := procstat.Percent(0, false)
	if err != nil {
		log.Fatal("could not get cpu percentages")
	}
	t := tstat[0]
	return []string{
		cpuStatToStr(t.Total()),
		cpuStatToStr(t.System),
		cpuStatToStr(t.User),
		cpuStatToStr(t.Idle),
		cpuStatToStr(t.Nice),
		cpuStatToStr(t.Iowait),
		cpuStatToStr(t.Irq),
		cpuStatToStr(t.Softirq),
		cpuStatToStr(t.Steal),
		cpuStatToStr(t.Guest),
		cpuStatToStr(pstat[0]),
	}
}
