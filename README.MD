# vip-keeper

Manages a virtual IP based on etcd

## 一、安装

1. $ wget https://github.com/wangjiaxi90/vipkeeper/releases/download/v0.0.5/vipkeeper_0.0.5_linux_amd64.tar.gz
2. $ tar -zxvf vipkeeper_0.0.5_linux_amd64.tar.gz
3. $ ./vipkeeper --version
## 二、帮助
```shell
$ ./vipkeeper --help
Usage of ./vipkeeper:
      --version            Show the version number.
      --ip string          Virtual IP address to configure.
      --netmask string     The netmask used for the IP address. Defaults to -1 which assigns ipv4 default mask.
      --interface string   Network interface to configure on .
      --endpoints string   Endpoint(s), separate multiple endpoints using commas. (default "http://127.0.0.1:2379" or "http://127.0.0.1:8500" depending on dcs-type.)
      --user string        Username for etcd DCS endpoints.
      --password string    Password for etcd DCS endpoints.
      --interval string    DCS scan interval in milliseconds. (default "1000")
```
## 三、参数说明

|参数名称|是否必须|样例|说明|
|----|----|----|----|
|version|否|--version|输出版本号|
|ip|是|--ip 192.168.176.252|绑定的VIP|
|netmask|是|--netmask 20|子网掩码的位数长度|
|interface|是|--interface eth0|VIP要绑定的网卡的设备名称|
|endpoints|否|--endpoints http://192.168.176.25:2379,http://192.168.176.121:2379|etcd的集群ip列表。默认：http://localhost:2379 注意：1. 多个ip之间英文逗号分割。2.http协议头别忘记。 3. 端口号别忘记。|
|user|否|--user root|登录etcd用到的用户名|
|password|否|--password 123456|登录etcd用到的密码|
|interval|否|--interval 1000|轮询的间隔时间。单位：毫秒。默认：1000。|

## 四、样例

```shell
$ ./vipkeeper --ip 192.168.176.252 --netmask 20 --interface eth0 --endpoints http://192.168.176.15:2379,http://192.168.176.121:2379,http://192.168.176.132:2379
```