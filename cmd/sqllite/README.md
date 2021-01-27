# sqlmetrics

```
Usage of sqlmetrics:
  -cleanup
    	cleanup database between insert sample (default true)
  -insertCount int
    	how many uuids to insert into the SqlLite database. (default 10000)
  -insertWait int
    	wait time between inserts in millisecond. (default 10)
  -metricsPort string
    	TCP port for metrics server to run from. (default ":8080")
  -recovery
    	remove database file before run. (default true)
  -sleepWait int
    	wait time for inserts to complete in seconds. (default 10)

```