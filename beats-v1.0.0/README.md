## filebeat-v1.0 源码学习
> - 官方文档： https://www.elastic.co/guide/en/beats/filebeat/1.0.1/_overview.html
> - GitHub： https://github.com/elastic/beats/blob/1.0.0/filebeat/beat/filebeat.go

### 日志收集流程
The basic model of execution:
- prospector 探测者: finds files in paths/globs to harvest 收割, starts harvesters 收割者
- harvester: reads a file, sends events to the spooler
- spooler: buffers events until ready to flush to the publisher 消费者
- publisher: writes to the network, notifies registrar
- registrar: records positions of files read

Finally, prospector uses the registrar information, on restart, to
determine where in each file to restart a harvester.

### filebeat-v1.0 架构图

![](https://www.elastic.co/guide/en/beats/filebeat/1.0.1/images/filebeat.png)


### filebeat 源码解析
- [filebeat源码解析](https://cloud.tencent.com/developer/article/1367784)
- [容器日志采集利器：filebeat深度剖析与实践](https://segmentfault.com/a/1190000019714761)
- [filebeat 源码分析](https://segmentfault.com/a/1190000006124064?utm_source=tag-newest)


#### 配置文件
- [filebeat 配置文件](http://s0www0elastic0co.icopy.site/guide/en/beats/filebeat/1.0.1/configuration-filebeat-options.html#_max_backoff)