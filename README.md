# aws_get_metric_statistics

aws_get_metric_statistics allows to collect AWS Cloudwatch data for the metrics VolumeWriteBytes, VolumeReadBytes, VolumeWriteOps, VolumeReadOps
of specified EBS volume and specified time span that can be more then 1440 data points with 1 hour granularity. Results are saved to CSV files.
Dates must be specified in the format 'YYYY-MM-DD'.

```console
$ go run aws_get_metric_statistics.go 
Some of mandatory parameter is absent: ["vol" "dates" "datee"]
aws_get_metric_statistics gathers disk metrics of AWS EBS volume. 'aws' command line tool must be setted up before. Dates must be specified in format YYYY-MM-DD.
Usage: aws_get_metric_statistics -vol vol-id -dates YYYY-MM-DD -datee YYYY-MM-DD
  -datee string
    	End date for gathered data
  -dates string
    	Start date for gathered data 
  -vol string
    	VolumeId of AWS EC2's EBS disk
Example
  aws_get_metric_statistics -vol vol-01234567890123456 -dates 2020-01-01 -datee 2020-03-04
exit status 1
```
