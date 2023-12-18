# apiserver 

`apiserver` provides comprehensive apis to work with [portal](https://github.com/kubeagi/portal). 

## Build

At the root dir of the project,run

```shell
make build-apiserver
```

## Run

```shell
(base) ➜  arcadia git:(main) ✗ ./bin/apiserver -h
Usage of ./bin/apiserver:
  -kubeconfig string
        Paths to a kubeconfig. Only required if out-of-cluster.
  -enable-playground
        enable the graphql playground
  -playground-endpoint-prefix string
        this parameter should also be configured when the service is forwarded via ingress and a path prefix is configured to avoid not finding the service, such as /apis
  -host string
        bind to the host, default is 0.0.0.0
  -port int
        service listening port (default 8081)
  -enable-oidc
        enable oidc authorization
  -client-id string
        oidc client id(required when enable odic)
  -client-secret string
        oidc client secret(required when enable odic)
  -data-processing-url string
        url to access data processing server (default "http://127.0.0.1:28888")
  -issuer-url string
        oidc issuer url(required when enable odic)
  -add_dir_header
        If true, adds the file directory to the header of the log messages
  -alsologtostderr
        log to standard error as well as files
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -log_file string
        If non-empty, use this log file
  -log_file_max_size uint
        Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
  -logtostderr
        log to standard error instead of files (default true)
  -master-url string
        k8s master url(required when enable odic)
  -one_output
        If true, only write logs to their native severity level (vs also writing to each lower severity level)
  -skip_headers
        If true, avoid header prefixes in the log messages
  -skip_log_headers
        If true, avoid headers when opening log files
  -stderrthreshold value
        logs at or above this threshold go to stderr (default 2)
  -system-namespace string
        system namespace where kubeagi has been installed
  -v value
        number for the log level verbosity
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```