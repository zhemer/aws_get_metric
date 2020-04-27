# aws_get_metric

aws_get_metric gathers AWS CloudWatch metrics specified by -ns, -name, -value and -metric parametrs and time span that can be more then 1440 data points with 1 hour granularity (more than 60 days).
Then the -cli command line switch is specified 'aws' CLI tool will be used and must be installed before.
Results are printed to console or may be saved to CSV file with specified or default file name.
Dates must be specified in the format 'YYYY-MM-DD'.

```console
$ go run aws_get_metric.go 
Some of mandatory parameter is absent: ["name" "times" "timee" "metric" "default" "ofile" "ns"]
aws_get_metric gathers AWS CloudWatch metrics specified by -namespace, -name and -metric-name parametrs (see example below).
'aws' command line tool must be setted up before. Dates must be specified in format YYYY-MM-DD.
Version 0.2.1
Usage: aws_get_metric -name <type> -value <value> -times YYYY-MM-DD -timee YYYY-MM-DD -metric <metric> [-ns <name>] [-ofile <file>]
  -cli
      Use command line utility 'aws' instead of native SDK to interface with AWS
  -debug
      Enable debbuging
  -metric string
      Metric's name
  -name string
      Object type to gather data, for example 'InstanceId' for EC2 instance
  -ns string
      Metric's namespace (default "EC2")
  -ofile string
      Write data to file, value 'default' generates default file name
  -timee string
      End date for gathered data period
  -times string
      Start date for gathered data period
  -value string
      Object's value to gather data, for example 'i-012345' for EC2 instance
Example
aws_get_metric -name InstanceId -value i-012345678 -times 2019-12-30 -timee 2020-01-02 -metric NetworkOut
```
