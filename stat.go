package main

import (
	"strconv"
	"time"

	sysstat "bitbucket.org/bertimus9/systemstat"
)

type StatSample struct {
	cpu sysstat.ProcCPUSample
	mem sysstat.MemSample
}

type MemAverage struct {
	Buffers   float64
	Cached    float64
	MemTotal  float64
	MemUsed   float64
	MemFree   float64
	SwapTotal float64
	SwapUsed  float64
	SwapFree  float64
	Duration  time.Duration
}

type StatAverage struct {
	cpu sysstat.ProcCPUAverage
	mem MemAverage
}

func programUptime() time.Duration {
	return time.Now().Sub(programEntryTime)
}

func statMemAvg(stat1 usize, stat2 usize) float64 {
	return (float64(stat1) + float64(stat2)) / 2.0
}

func memAverage(s1, s2 *sysstat.MemSample) MemAverage {
	return MemAverage{
		statMemAvg(s1.Buffers, s2.Buffers),
		statMemAvg(s1.Cached, s2.Cached),
		statMemAvg(s1.MemTotal, s2.MemTotal),
		statMemAvg(s1.MemUsed, s2.MemUsed),
		statMemAvg(s1.MemFree, s2.MemFree),
		statMemAvg(s1.SwapTotal, s2.SwapTotal),
		statMemAvg(s1.SwapUsed, s2.SwapUsed),
		statMemAvg(s1.SwapFree, s2.SwapFree),
		s2.Time.Sub(s1.Time),
	}
}

func statSample() StatSample {
	cpu := sysstat.GetProcCPUSample()
	mem := sysstat.GetMemSample()
	return StatSample{cpu, mem}
}

func (s1 *StatSample) averageToNow() StatAverage {
	s2 := statSample()
	return StatAverage{
		sysstat.GetProcCPUAverage(s1.cpu, s2.cpu, programUptime().Seconds()),
		memAverage(&s1.mem, &s2.mem),
	}
}

func statToStr(stat float64) string {
	return strconv.FormatFloat(stat, 'f', 10, 64)
}

func (s *StatAverage) CsvHeader() []string {
	return []string{
		"proc.cpu.pct.user",
		"proc.cpu.pct.system",
		"proc.cpu.pct.total",
		"proc.cpu.pct.cumulativeTotal",
		"proc.cpu.pct.possible",

		"sys.mem.buffers",
		"sys.mem.cached",
		"sys.mem.memTotal",
		"sys.mem.memUsed",
		"sys.mem.memFree",
		"sys.mem.swapTotal",
		"sys.mem.swapUsed",
		"sys.mem.swapFree",

		"time.stamp",
		"time.duration.ms",
	}
}

func (s *StatAverage) ToCsvRow() []string {
	c := s.cpu
	m := s.mem
	return []string{
		statFToStr(c.UserPct),
		statFToStr(c.SystemPct),
		statFToStr(c.TotalPct),
		statFToStr(c.CumulativeTotalPct),
		statFToStr(c.PossiblePct),

		statFToStr(m.Buffers),
		statFToStr(m.Cached),
		statFToStr(m.MemTotal),
		statFToStr(m.MemUsed),
		statFToStr(m.MemFree),
		statFToStr(m.SwapTotal),
		statFToStr(m.SwapUsed),
		statFToStr(m.SwapFree),

		statIToStr(c.Time.Unix()),
		statFToStr(c.Seconds * 1000),
	}
}

func statFToStr(stat float64) string {
	return strconv.FormatFloat(stat, 'f', 10, 64)
}

func statIToStr(stat int64) string {
	return strconv.FormatInt(stat, 10)
}
