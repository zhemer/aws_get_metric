package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type tmStrToStr map[string][]string

const sVersion = "0.2.0"
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
	iCli       = flag.Bool("cli", false, "Use command line utility 'aws' instead of native SDK to interface with AWS")
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
		os.Exit(0)
	}

	aaDateToVal := make(tmStrToStr)
	timeSt, _ := time.Parse(time.RFC3339, *sDateSt+sTimeRfc3339)
	timeEn, _ := time.Parse(time.RFC3339, *sDateEn+sTimeRfc3339)
	tDiffHours := timeEn.Sub(timeSt).Hours()
	if *iDbg {
		fmt.Printf("ds=%q de=%q diff=%q\n", timeSt, timeEn, tDiffHours)
	}
	if !*iCli {
		awsGetMetricsData(*sName, *sMetric, *sNamespace, timeSt, timeEn)
		os.Exit(0)
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
			lines := awsGetMetricStatistics0(*sName, t0, t1, *sMetric, *sNamespace)
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
		f.WriteString(fmt.Sprintf("%s,%s\n", keyTime, aaDateToVal[keyTime][0]))
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

func getCPUmodel() string {
	cmd := "cat /proc/cpuinfo | egrep '^model name' | uniq | awk '{print substr($0, index($0,$4))}'"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Sprintf("Failed to execute command: %s", cmd)
	}
	return string(out)
}
func awsGetMetricStatistics0(sName, sDateSt, sDateEn, sMetric string, sNamespace string) []string {
	sCmd := "aws cloudwatch get-metric-statistics --period 3600 --statistics Maximum --dimensions Name=%s --start-time %s --end-time %s --output text --metric-name %s --namespace AWS/%s|sed 's/T/ /g'|sed 's/Z//'"
	sCmd1 := fmt.Sprintf(sCmd, sName, sDateSt, sDateEn, sMetric, sNamespace)
	if *iDbg {
		fmt.Printf("Running command: %q\n", sCmd1)
	}
	out, err := exec.Command("bash", "-c", sCmd1).Output()
	if err != nil {
		log.Fatalf("exec.Command() failed with: %q\n", err)
	}
	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")[1:]
}
func awsGetMetricStatistics(sName, sDateSt, sDateEn, sMetric string, sNamespace string) []string {
	var sCmdAws = "aws"
	var sCmd = "cloudwatch get-metric-statistics --period 3600 --statistics Maximum --dimensions Name=%s --start-time %s --end-time %s --output text --metric-name %s --namespace AWS/%s"
	sCmd1 := fmt.Sprintf(sCmd, sName, sDateSt, sDateEn, sMetric, sNamespace)
	if *iDbg {
		fmt.Printf("Running command: '%s %s'\n", sCmdAws, sCmd1)
	}
	cmd := exec.Command(sCmdAws, strings.Split(sCmd1, " ")...)
	var outErr bytes.Buffer
	cmd.Stderr = &outErr
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("cmd.Run() failed with: %s: %s\n", err, outErr.String())
	}

	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")[1:]
}

func awsGetMetricsData(sName, sMetric, sNamespace string, startTime, endTime time.Time) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := cloudwatch.New(sess)

	// fmt.Printf("%v %v\n", startTime, endTime)
	namespace := "AWS/" + sNamespace
	metricname := sMetric
	metricid := "m1"
	asNameVal := strings.Split(strings.ReplaceAll(sName, "=", ","), ",")
	// fmt.Printf("\nasNameVal=%v\n", asNameVal)
	metricDim1Name := asNameVal[0]
	metricDim1Value := asNameVal[2]
	// metricDim2Name := "DomainName"
	// metricDim2Value := "yourdomainhere"
	period := int64(3600)
	stat := "Maximum"
	query := &cloudwatch.MetricDataQuery{
		Id: &metricid,
		MetricStat: &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Namespace:  &namespace,
				MetricName: &metricname,
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  &metricDim1Name,
						Value: &metricDim1Value,
					},
					// {
					// 	Name:  &metricDim2Name,
					// 	Value: &metricDim2Value,
					// },
				},
			},
			Period: &period,
			Stat:   &stat,
		},
	}

	resp, err := svc.GetMetricData(&cloudwatch.GetMetricDataInput{
		EndTime:           &endTime,
		StartTime:         &startTime,
		MetricDataQueries: []*cloudwatch.MetricDataQuery{query},
	})

	if err != nil {
		fmt.Println("Got error getting metric data")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Printf("date,%s\n", sMetric)
	for _, metricdata := range resp.MetricDataResults {
		// fmt.Println(*metricdata.Id)
		for index := range metricdata.Timestamps {
			// fmt.Printf("%v,%v\n", (*metricdata.Timestamps[index]).String(), *metricdata.Values[index])
			fmt.Printf("%v,%v\n", (*metricdata.Timestamps[index]).Format("2006-01-02 15:04:05"), *metricdata.Values[index])
		}
	}
}
