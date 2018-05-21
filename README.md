用于反向代理 HTTP 后端接口, 并允许额外设置 origin 再次跨域许可

```
Usage of ./bin/reverse_http_proxy:
  -bind string
        此代理程序监听的地址端口 (default ":8080")
  -help
        显示此帮助信息
  -log string
        重定向日志输出到指定文件 (发送 SIGUSR2 信号重载)
  -origin value
        允许通过进行透明代理的源域名后缀, 设置或替换代理目标 CORS 配置 (Access-Control-Allow-Origin), 可进行多次指定; 默认允许所有源域名跨域请求;
  -suffix value
        允许通过进行透明代理的目标域名后缀, 可进行多次指定; 默认允许所有域名通过;
```
