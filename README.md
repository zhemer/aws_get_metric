# aws_get_metric_statistics

aws_get_metric_statistics gathers AWS CloudWatch metrics specified by -namespace, -name and -metric-name parametrs and time span that can be more then 1440 data points with 1 hour granularity (more than 60 days).
Results are printed to console or may be saved to CSV file with specified or default file name.
Dates must be specified in the format 'YYYY-MM-DD'.

```console
$ ./aws_get_metric_statistics
aws_get_metric_statistics gathers AWS CloudWatch metrics specified by -namespace, -name and -metric-name parametrs for specified time span (see example below).
'aws' command line tool must be setted up before. Dates must be specified in format YYYY-MM-DD.
Version 0.0.1
Usage: aws_get_metric_statistics -name <object type>,Value=<value> -start-time YYYY-MM-DD -end-time YYYY-MM-DD -metric-name <metric> -namespace <name>
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
  aws_get_metric_statistics -name InstanceId,Value=i-012345 -start-time 2019-11-29 -end-time 2020-03-04 -metric-name NetworkOut
```
