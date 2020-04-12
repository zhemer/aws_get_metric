package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type tmStrToStr map[string][]string

const sVersion = "0.1.0"
const iDataPointLimit = 1440 // number of datapoint (or hours) with 3600 period that equals 60 days
const sTimeRfc3339 = "T00:00:00Z"

var aParams = []string{"name", "start-time", "end-time", "metric-name", "default"}
var (
	sName      = flag.String(aParams[0], "", "Object type/value to gather data, for example 'InstanceId,Value=i-012345' for EC2 instance")
	sDateSt    = flag.String(aParams[1], "", "Start date for gathered data period")
	sDateEn    = flag.String(aParams[2], "", "End date for gathered data period")
	sMetric    = flag.String(aParams[3], "", "Metric's name")
	sOutFile   = flag.String("out-file", "", "Write data to file, value 'default' generates default file name")
	sNamespace = flag.String("namespace", "EC2", "Metric's namespace")
	iDbg       = flag.Bool("debug", false, "Enable debbuging")
)
var sHelp = `Example
  %s -name InstanceId,Value=i-012345 -start-time 2019-11-29 -end-time 2020-03-04 -metric-name NetworkOut`

var aStrTable = []string{
	`%s gathers AWS CloudWatch metrics specified by -namespace, -name and -metric-name parametrs (see example below).
'aws' command line tool must be setted up before. Dates must be specified in format YYYY-MM-DD.
Version %s` + "\n",
	"Some of mandatory parameter is absent: %q\n"}

func main() {
	flag.Usage = func() {
		path := strings.Split(os.Args[0], "/")
		cmd := path[len(path)-1]
		fmt.Fprintf(flag.CommandLine.Output(), aStrTable[0], cmd, sVersion)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s -name <object type>,Value=<value> -start-time YYYY-MM-DD -end-time YYYY-MM-DD -metric-name <metric> -namespace <name>\n", cmd)
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), sHelp+"\n", cmd)
	}
	flag.Parse()
	if *sName == "" || *sDateSt == "" || *sDateEn == "" || *sMetric == "" {
		fmt.Printf(aStrTable[1], aParams)
		flag.Usage()
		os.Exit(1)
	}

	aaDateToVal := make(tmStrToStr)
	timeSt, _ := time.Parse(time.RFC3339, *sDateSt+sTimeRfc3339)
	timeEn, _ := time.Parse(time.RFC3339, *sDateEn+sTimeRfc3339)
	tDiffHours := timeEn.Sub(timeSt).Hours()
	if *iDbg {
		fmt.Printf("ds=%q de=%q diff=%q\n", timeSt, timeEn, tDiffHours)
	}

	iCnt := int(tDiffHours) / iDataPointLimit
	if int(tDiffHours)%iDataPointLimit > 0 {
		iCnt++
	}

	chan0 := make(chan tmStrToStr, iCnt)
	defer func() {
		close(chan0)
	}()

	if *iDbg {
		fmt.Printf("iCnt=%q tDiffHours=%q iDataPointLimit=%q\n", iCnt, tDiffHours, iDataPointLimit)
	}
	t0 := timeSt
	t1 := timeSt.AddDate(0, 0, 60)
	for i := 0; i < iCnt; i++ {
		if t1.After(timeEn) {
			t1 = timeEn
			if t0 == t1 {
				t0 = t0.AddDate(0, 0, -1)
			}
		}
		if *iDbg {
			fmt.Printf("t0=%s t1=%s\n", t0.String()[:10], t1.String()[:10])
		}

		go func(t0 string, t1 string, i int) {
			lines := awsGetMetricStatistics(*sName, t0, t1, *sMetric, *sNamespace)
			if *iDbg {
				// fmt.Printf("lines=%v ...\n", lines[0:2])
				fmt.Printf("started %d\n", i)
			}
			chan0 <- tmStrToStr{strconv.Itoa(i): lines}
		}(t0.String()[:10], t1.String()[:10], i)
		t0 = t1.AddDate(0, 0, 1)
		t1 = t0.AddDate(0, 0, 60)
	}

	doneCount := 0
	for doneCount != iCnt {
		select {
		case chunk := <-chan0:
			for i := range chunk {
				awsOutToArray(chunk[i], aaDateToVal)
				doneCount++
				if *iDbg {
					fmt.Printf("ended %s\n", i)
				}

			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	sFilename := ""
	f := os.Stdout
	if *sOutFile != "" {
		sFilename = *sOutFile
		if *sOutFile == aParams[4] {
			sFilename = *sNamespace + "-" + *sMetric + "-" + *sName
		}
		// sFilename = sFilename + ".csv"
		f, _ = os.Create(sFilename)
		defer f.Close()
	}
	f.WriteString(fmt.Sprintf("date,%s\n", *sMetric))
	// fmt.Printf("aaDateToVal=%v\n", aaDateToVal)
	// Sorting keys(dates) of the aaDateToVal
	keys := make([]string, 0, len(aaDateToVal))
	for k := range aaDateToVal {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, keyTime := range keys {
		t1 := keyTime[:len(keyTime)-1]
		t1 = strings.ReplaceAll(t1, "T", " ")
		// if len(aaDateToVal[t]) == 1 {
		// 	aaDateToVal[t] = append(aaDateToVal[t], "-1")
		// }
		f.WriteString(fmt.Sprintf("%s,%s\n", t1, aaDateToVal[keyTime][0]))
	}
	if *sOutFile != "" {
		fmt.Printf("Metric '%s/%s' data with name %q were saved to the file %q\n", *sNamespace, *sMetric, *sName, sFilename)
	}

}

func awsOutToArray(out []string, arr tmStrToStr) {
	for l := range out {
		line := out[l]
		// fmt.Printf("%d %s\n", l, line)
		vals := strings.Split(line, "\t")
		// fmt.Printf("vals=%q\n", vals)
		if vals[1] == "" {
			vals[1] = "-1"
		}
		arr[vals[2]] = append(arr[vals[2]], vals[1])
	}
}

func awsGetMetricStatistics(sName, sDateSt, sDateEn, sMetric string, sNamespace string) []string {
	var sCmdAws = "aws"
	var sCmd = "cloudwatch get-metric-statistics --period 3600 --statistics Maximum --dimensions Name=%s --start-time %s --end-time %s --output text --metric-name %s --namespace AWS/%s"
	sCmd1 := fmt.Sprintf(sCmd, sName, sDateSt, sDateEn, sMetric, sNamespace)
	if *iDbg {
		fmt.Printf("Running command: '%s %s'\n", sCmdAws, sCmd1)
	}
	// return nil
	cmd := exec.Command(sCmdAws, strings.Split(sCmd1, " ")...)
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	// out, err := cmd.CombinedOutput()
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("cmd.Run() failed with: %s: %s\n", err, outErr.String())
	}

	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")[1:]
	// return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")[1:5]
}
