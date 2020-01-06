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
func main() {
	logrus.SetFormatter(&selfformatter.EaseFormatter{
		Formatter:                 "%time% %level% [%kv%] -- %msg%",
		KvCom:                     "=",
		FieldMapCom:               "&",
		ForceColors:               false,
		DisableColors:             false,
		EnvironmentOverrideColors: false,
		DisableTimestamp:          false,
		FullTimestamp:             true,
	})
	logrus.SetLevel(logrus.TraceLevel)
	logrus.WithField("instanceName", "logrus").WithField("attr1", "kafka").Error("hello easy formatter")
	logrus.WithField("instanceName", "logrus").WithField("attr2", "rabbit").Info("hello easy formatter")
	logrus.WithField("instanceName", "logrus").WithField("attr5", "rocket").Debug(nil)
	logrus.WithField("instanceName", "logrus").WithField("attr5", "rocket").Trace("hello easy formatter")
}
```

得到如下输出:

![formatter](https://github.com/anypick/image-storage/blob/master/logrus-self-formatter-01.png)

# Detail

`Formatter`: 定义日志输出的格式，目前包含9个可替换字段。分别是四个必须字段，和五个弹性字段。

如定一个任意格式的日志：

```shell
%time% %level% [%attr1%] [%kv%] -- %msg%
```

1. 四个必须字段：

   `%time%`: 日志时间

   `%level%`: 日志级别

   `%kv%`:日志中Fields, 如:

   ```go
   logurs.WithFields("key", "value")
   ```

   `"%msg%"`: 日志消息

2. 五个弹性字段：

`%attr1%`,`%attr2%`,`%attr3%`,`%attr4%`,`%attr5%`, 开发者在开发的过程中会有一些不确定的属性字段进行定义，所以在这里保留了五个弹性字段。用法如下：

```go
logurs.WithFields("attr1", "attr1_value")
```

`KvCom`: kv直接的连接方式，在Quik start中使用的是=

`FieldMapCom`：多个属性字段之间的连接方式，在Quik start中使用的是&



