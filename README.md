# AI写的易于部署的探针（至少我是那么认为）
  因为我的脑容量不足以支持我安装哪吒探针，所以我花了大概4个小时用Ai写了这个玩意，反正部署简单就完事了，至于美观？自己写html吧

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

系统要求：Debian12/Ubuntu22.04+，AMD64架构，至少一个vCPU，1GB RAM

亮点：部署简单

缺点：丑，安全性未知，功能较差，依赖安装时的一次性配置

免责声明
```text
软件许可及免责声明 / Software License & Disclaimer
该软件以GPLv3许可证开源
本程序按“原样”（"AS IS"）提供，不附带任何形式的明示或暗示保证。作者不保证程序符合特定用途，亦不保证运行过程中不出现错误。
在任何情况下，作者不对因使用本程序产生的任何损害（包括数据丢失、系统崩溃、法律诉讼等）承担任何责任。作者的全部赔偿责任上限在任何情况下均不超过用户实际支付的授权费用（如有）。
用户一旦运行、调试或以任何方式使用本程序，即视为完全理解并接受上述条款。作者保留对本协议的最终解释权，并有权随时更新授权条款。
```
~~又刷了贡献值~~ (bushi

安装
```bash
bash <(curl -sL https://a057.net/dows/tanzhen/install.sh | tr -d '\r')
```
[长这样](https://a05.uk/z)

### 更新日志：v0.02BETA

0.~~又刷了贡献值~~ (bushi

1.新增 /api 路径，返回JSON格式数据，如下

```text
[
  {
    "id": 1,
    "name": "传家宝 4c4g",
    "line": "CN2GIA/9929/CMIN2",
    "price": "12.25￥",
    "expiry": "月付",
    "cpu": 0,
    "ram": 26.9248515426976,
    "disk": 26.9983769121111,
    "swap": 0,
    "latency": 1,
    "last_update": "0001-01-01T00:00:00Z"
  },
  {
    "id": 2,
    "name": "RackNerd2026新年特别版",
    "line": "Cogent",
    "price": "11.29$",
    "expiry": "03/27/2027",
    "cpu": 0,
    "ram": 69.2663198113975,
    "disk": 36.1272622954784,
    "swap": 15.3728876060171,
    "latency": 1,
    "last_update": "0001-01-01T00:00:00Z"
  }
]
```

[这里](https://tanzhen.a057.net/api)

2.更新默认html页面

 1.增加Swap占用显示
 
 2.每项资源显示详细占用数值
 

新安装的获取的是最新的版本

之前安装过的需要手动去

`https://node1-rn.a05777.uk:8443/dows/tanzhen/server-bin`

下载新版本并且替换，客户端未改变

新版html

`https://node1-rn.a05777.uk:8443/dows/tanzhen/jiankong.html`

旧版html&服务端

`https://node1-rn.a05777.uk:8443/dows/tanzhen/server-bin2`

`https://node1-rn.a05777.uk:8443/dows/tanzhen/jiankong.html2`


开发经历：用Claude想偷懒，把代码丢给claude，没想到免费版额度不够，还是用的Gemini，上传新的服务端的时候SFTP掉了3次，编辑html的时候SSH掉了一次，哪家运营商：世界加钱可及
