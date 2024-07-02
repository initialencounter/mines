# mines
多人扫雷服务器

[即刻体验](http://14.103.48.1:2330/)

## 项目架构

- 后端 golang-fiber
- 前端 Vue

## 工作原理

后端负责扫雷的逻辑以及用户的管理
玩家首次打开网页需要输入账号和密码，默认账号密码是

账号:`john` 密码:`doe`

前端负责展示游戏界面以及与后端的交互
前后端交互采用 http + websocket 的方式通信

登录，获取地图信息采用 http 请求

扫雷地图会通过服务端websocket来更新，玩家的操作也会通过websocket来传递给服务端

## 部署

前往 [GitHub Action](https://github.com/initialencounter/mines/actions/runs/9763654252) 下载编译好的二进制文件，然后运行即可

## TODO

目前还有很多功能没有实现，例如

- 排行榜
- 成绩计算分析
- 扫雷道具
- 后端数据库
- 等等

欢迎大家来给我提 PR 呀
[开源地址](https://github.com/initialencounter/mines)