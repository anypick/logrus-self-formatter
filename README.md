# logrus-self-formatter

logrus-self-formatter是logrus的一种日志格式，它允许你自由定义格式。如：`%time% [%level%] %msg% %kv%`将会得到如下日志输出:

```shell
2020-01-06T17:41:40+08:00 [ERRO] request success type=redis&requestUri=/ping
```

## Quick Start

```shell
$ go get github.com/anypick/logrus-self-formatter
```

**【main.go】**

```go
logrus.SetFormatter(&selfformatter.EaseFormatter{
		Formatter:                 "%time% %level% [%attr1%] [%attr2%] [%attr5%] [%kv%] -- %msg%",
		KvCom:                     "=",
		FieldMapCom:               "&",
		ForceColors:               false,
		DisableColors:             false,
		EnvironmentOverrideColors: false,
		DisableTimestamp:          false,
		FullTimestamp:             true,
	})
```







# Detail

