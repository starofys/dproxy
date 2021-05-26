## dproxy隧道代理服务器

> HTTP/SOCKET5 proxy over SSH.

### 功能特性

- [x] SSH隧道
- [x] HTTP代理
- [x] SOCKET代理
- [x] 端口重定向转发
- [x] 反向代理(仅http代理)


### 配置

```yaml
id_rsa: 'E:/rsa_id'
# socket 监听地址
local_socket: '127.0.0.1:1080'
# http 监听地址
local_normal: '127.0.0.1:1316'
remote: 'ssh://user@10.227.112.9:22223'
# ssh 隧道配置，对于不支持代理的服务使用下面的配置
tunnel:
  - name: 开发mysql
    local: '127.0.0.1:12120'
    remote: '10.227.100.2:3006'

  - name: 测试mysql
    local: '127.0.0.1:12121'
    remote: '10.227.100.4:3006'

  - name: 开发redis
    local: '127.0.0.1:6380'
    remote: '10.227.100.20:8039'

  - name: 测试redis
    local: '127.0.0.1:3036'
    remote: '10.227.96.8:3036'

  - name: Sonar
    local: '127.0.0.1:8000'
    remote: '10.227.100.22:8000'
# tcp 端口转发配置，http转发建议使用 proxy_pass 反向代理
forward:
# 反向代理配置
proxy_pass:
  www.test.com:
    /bssp/api/:
      target: 'http://localhost:8058/'
# 只有列表内的地址通过代理隧道
proxy:
  - 10.227.*.*
```

### 配置介绍

|配置| 描述 |
|----|----|
|`local_socket`|**socket代理端口**|
|`local_normal`|**本地http代理端口**|
|`remote`|**目标ssh服务器或者跳板机**|
|`tunnel`|**ssh隧道映射(对于不支持代理访问的应用可以使用)**|
|`forward`|**端口转发**|
|`local_normal`|**本地http代理端口**|
|`proxy_pass`|**反向代理配置**|
|`proxy`|**允许使用的代理列表**|



### 隧道数组格式示例

|本地监听地址 | 目标地址 | 备注 |
| ---- | ---- | ---- |
|127.0.0.1:8000|10.227.100.2:3006|开发mysql


### 反向代理示例

```yaml
proxy_pass:
  www.test.com:
    /bssp/api/:
      target: 'http://localhost:8058/'
```




