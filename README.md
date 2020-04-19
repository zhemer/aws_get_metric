# aws_get_metric

aws_get_metric gathers AWS CloudWatch metrics specified by -namespace, -name and -metric-name parametrs and time span that can be more then 1440 data points with 1 hour granularity (more than 60 days).
Then the -cli command line switch is specified 'aws' CLI tool must be installed.
Results are printed to console or may be saved to CSV file with specified or default file name.
Dates must be specified in the format 'YYYY-MM-DD'.

```console
$ ./aws_get_metric
aws_get_metric gathers AWS CloudWatch metrics specified by -namespace, -name and -metric-name parametrs (see example below).
'aws' command line tool must be setted up before in case of using -cli command line switch. Dates must be specified in format YYYY-MM-DD.
Version 0.2.0
Usage: aws_get_metric -name <object type>,Value=<value> -start-time YYYY-MM-DD -end-time YYYY-MM-DD -metric-name <metric> -namespace <name>
  -cli
      Use command line utility 'aws' instead of native SDK to interface with AWS
  -debug
      Enable debbuging
  -end-time string
      End date for gathered data period
  -metric-name string
      Metric's name
  -name string
      Object type/value to gather data, for example 'InstanceId,Value=i-012345' for EC2 instance
  -namespace string
      Metric's namespace (default "EC2")
  -out-file string
      Write data to file, value 'default' generates default file name
  -start-time string
      Start date for gathered data period
Example
  aws_get_metric -name InstanceId,Value=i-012345 -start-time 2019-11-29 -end-time 2020-03-04 -metric-name NetworkOut

```
