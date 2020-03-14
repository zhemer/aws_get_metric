package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type aaStrToFloat map[string][]float64
type aaStrToInt map[string][]int64

const iDataPointLimit = 1440 // number of datapoint (or hours) with 3600 period that equals 60 days
const sTimeRfc3339 = "T00:00:00Z"

var aParams = []string{"vol", "dates", "datee"}
var (
	sVol    = flag.String(aParams[0], "", "VolumeId of AWS EC2's EBS disk")
	sDateSt = flag.String(aParams[1], "", "Start date for gathered data ")
	sDateEn = flag.String(aParams[2], "", "End date for gathered data")
)
var sHelp = `Examples
  %s -vol vol-01234567890123456 -dates 2020-03-01 -datee 2020-03-04`

var aStrTable = []string{
	"%s gathers disk metrics of AWS EBS volume. 'aws' command line tool must be setted up before. Dates must be specified in format YYYY-MM-DD.\n",
	"Some of mandatory parameter is absent: %q\n"}

func main() {
	flag.Usage = func() {
		path := strings.Split(os.Args[0], "/")
		cmd := path[len(path)-1]
		fmt.Fprintf(flag.CommandLine.Output(), aStrTable[0], cmd)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s -vol vol-id -dates YYYY-MM-DD -datee YYYY-MM-DD\n", cmd)
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), sHelp+"\n", cmd)
	}
	flag.Parse()
	if *sVol == "" || *sDateSt == "" || *sDateEn == "" {
		fmt.Printf(aStrTable[1], aParams)
		flag.Usage()
		os.Exit(1)
	}

	timeSt, _ := time.Parse(time.RFC3339, *sDateSt+sTimeRfc3339)
	timeEn, _ := time.Parse(time.RFC3339, *sDateEn+sTimeRfc3339)
	tDiffHours := timeEn.Sub(timeSt).Hours()
	fmt.Printf("ds=%q de=%q diff=%q\n", timeSt, timeEn, tDiffHours)

	aVolRWBytes := make(aaStrToInt)
	aVolRWOps := make(aaStrToInt)
	// fmt.Printf("aVolRWBytes=%v\n", aVolRWBytes)

	iCnt := int(tDiffHours) / iDataPointLimit
	if int(tDiffHours)%iDataPointLimit > 0 {
		iCnt++
	}
	// fmt.Printf("iCnt=%q tDiffHours=%q iDataPointLimit=%q\n", iCnt, tDiffHours, iDataPointLimit)
	t0 := timeSt
	t1 := timeSt.AddDate(0, 0, 60)
	for i := 0; i < iCnt; i++ {
		if t1.After(timeEn) {
			t1 = timeEn
		}
		// fmt.Printf("t0=%s t1=%s\n", t0.String()[:10], t1.String()[:10])

		lines := aws_get_metric_statistics(*sVol, t0.String()[:10], t1.String()[:10], "VolumeWriteBytes")
		out_to_array(lines, aVolRWBytes)
		lines = aws_get_metric_statistics(*sVol, t0.String()[:10], t1.String()[:10], "VolumeReadBytes")
		out_to_array(lines, aVolRWBytes)

		lines = aws_get_metric_statistics(*sVol, t0.String()[:10], t1.String()[:10], "VolumeWriteOps")
		out_to_array(lines, aVolRWOps)
		lines = aws_get_metric_statistics(*sVol, t0.String()[:10], t1.String()[:10], "VolumeReadOps")
		out_to_array(lines, aVolRWOps)

		t0 = t1.AddDate(0, 0, 1)
		t1 = t0.AddDate(0, 0, 60)
	}

	sName := *sVol + "-VolumeBytes.csv"
	f, _ := os.Create(sName)
	defer f.Close()
	f.WriteString(fmt.Sprintf("date,VolumeWriteBytes,VolumeReadBytes\n"))
	for t := range aVolRWBytes {
		t1 := t[:len(t)-1]
		t1 = strings.ReplaceAll(t1, "T", " ")
		f.WriteString(fmt.Sprintf("%s,%d,%d\n", t1, aVolRWBytes[t][0], aVolRWBytes[t][1]))
	}
	fmt.Printf("Writed file %q\n", sName)

	sName = *sVol + "-VolumeOps.csv"
	f, _ = os.Create(sName)
	f.WriteString(fmt.Sprintf("date,VolumeWriteOps,VolumeReadOps\n"))
	defer f.Close()
	for t := range aVolRWOps {
		t1 := t[:len(t)-1]
		t1 = strings.ReplaceAll(t1, "T", " ")
		// fmt.Printf("%s,%f,%f\n", t1, aVolRWOps[t][0], aVolRWOps[t][1])
		f.WriteString(fmt.Sprintf("%s,%d,%d\n", t1, aVolRWOps[t][0], aVolRWOps[t][1]))
	}
	fmt.Printf("Writed file %q\n", sName)
}

func out_to_array(out []string, arr aaStrToInt) {
	for l := range out {
		line := out[l]
		// fmt.Printf("%d %s\n", l, line)
		vals := strings.Split(line, "\t")
		// fmt.Printf("vals=%q\n", vals)
		fVal, _ := strconv.ParseFloat(vals[1], 64)
		arr[vals[2]] = append(arr[vals[2]], int64(fVal))
	}

}

func aws_get_metric_statistics(sVol, sDateSt, sDateEn, sMetric string) []string {
	var sCmdAws = "aws"
	var sCmd = "cloudwatch get-metric-statistics --period 3600 --namespace AWS/EBS --statistics Maximum --dimensions Name=VolumeId,Value=%s --start-time %s --end-time %s --output text --metric-name %s"
	sCmd1 := fmt.Sprintf(sCmd, sVol, sDateSt, sDateEn, sMetric)
	fmt.Printf("Running command: '%s %s'\n", sCmdAws, sCmd1)
	cmd := exec.Command(sCmdAws, strings.Split(sCmd1, " ")...)
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	// out, err := cmd.CombinedOutput()
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("cmd.Run() failed with: %s: %s\n", err, outErr.String())
	}

	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")[1:]
}
