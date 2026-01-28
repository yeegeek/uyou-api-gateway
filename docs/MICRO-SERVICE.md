## 一、微服务拆分方案

基于系统功能分析和性能要求，将系统拆分为以下9个微服务：

```
┌─────────────────────────────────────────────────────────────────┐


### 1. 中心服务（Core Service）

github仓库地址： https://github.com/yeegeek/uyou-core-service.git

**技术栈：** Go + PostgreSQL + Redis

**职责：**
- 用户注册，登陆，找回密码（可以简化为使用验证码登陆）
- 用户认证与授权（JWT Token生成、刷新、验证）
- 第三方登录（Facebook、微信）
- 用户资料管理（CRUD、推荐用户生成）
- 好友关系管理（请求、接受、拒绝、删除、分组、黑名单）
- 访客记录管理
- 支付系统（账户充值、付费图片购买、交易记录、提现）
- 商品与订单管理（商品CRUD、订单创建/查询/状态更新）
- 系统设置与公告管理

**定时任务：**

- 生成推荐用户/在线用户/最新用户/生日用户
- 获取汇率
- 订单自动发货/确认提醒
- 更新用户年龄/不活跃状态

---

### 2. 聊天服务（Chat Service）

github仓库地址： https://github.com/yeegeek/uyou-chat-service.git
**技术栈：** Go + MongoDB + Redis

**职责：**

主要功能： 

发送消息： 发送文字，图片，视频, 语音， 文件， 位置坐标，发红包（这个应该只是个标记， 还需要其他微服务处理）

（其中文字如果是超链接，  自动转化为网页预览或者如果是图片视频的话转化为图片视频）

撤回消息
引用回复某个消息
点赞某个消息， 
删除消息
清空聊天记录
定时清空聊天记录
消息的阅后即焚（这个功能可以前端实现，其实也就是对应删除接口）
音视频通话

获取所有对话列表（chats)
获取某个对话详情 (chat)
删除整个对话（清空消息内容）
不再显示某个对话（不清空消息内容）
标记某个对话未读
置顶某个对话
某个对话消息免打扰（不提示）
视频通话管理（创建、状态更新、结束处理、断线重连）

**定时任务：**

- 处理断线通话的信件发送

---

### 3. 动态服务（Post Service）
github仓库地址： https://github.com/yeegeek/uyou-post-service.git
**技术栈：** Go + MongoDB + Redis

**职责：**

- 动态发布/删除/查看
- 动态列表（全部/某用户）
- 点赞功能（点赞/取消/列表）
- 评论功能（可选）

---

### 4. 内容服务（Media Service）
github仓库地址： https://github.com/yeegeek/uyou-media-service.git
**技术栈：** Go + PostgreSQL + S3 + AWS Rekognition

**职责：**

- 图片/视频上传
- 收费图片管理
- 文件存储（AWS S3）
- 人脸识别检测（防止重复注册、盗图检测）
- 内容审核（调用AI服务检测敏感内容）
- 图片处理（裁剪、压缩，AWS Lambda）

**定时任务：**

- 上传图片进行人脸识别入库

---

### 5. 搜索服务（Search Service）
github仓库地址： https://github.com/yeegeek/uyou-search-service.git
**技术栈：** Go + Elasticsearch + Redis

**职责：**

- 用户搜索（多条件筛选）
- 地理位置搜索（附近的人）
- 全文搜索
- 搜索索引管理（用户索引同步）

---

### 6. WebSocket服务（WebSocket Service）
github仓库地址： https://github.com/yeegeek/uyou-websocket-service.git
**技术栈：** Go + Redis

**职责：**

- WebSocket连接管理（连接/断开/心跳/重连）
- 实时消息推送（新消息、翻译完成、支付成功等）
- 消息广播与单播
- 在线状态同步与查询
- 用户踢下线

---

### 7. 通知服务（Notification Service）
github仓库地址： https://github.com/yeegeek/uyou-notification-service.git
**技术栈：** Go + PostgreSQL + Redis + Mailgun/SES

**职责：**

- 通知历史记录管理
- 通知模板管理
- **推送通知：**
  - WebSocket实时推送（调用WebSocket服务）
  - Firebase移动端推送
- **邮件通知：**
  - 邮件发送（Mailgun/SES）
  - 邮件模板管理（验证码、消息通知、营销邮件等）
  - Mailgun Webhooks处理（退回、投诉）
- **短信通知：**
  - 短信验证码发送（云片）

**定时任务：**

- 邮件提醒（长期不登录用户）
- 处理邮件发送队列

---

### 8. AI服务（AI Service）
github仓库地址： https://github.com/yeegeek/uyou-ai-service.git
**技术栈：** Go + Redis

**职责：**

- AI翻译接口（Google Translate，支持8种语言）
- AI消息建议（OpenAI GPT）
- 用户行为分析/AI检测
- AI内容检测（敏感内容识别）
- 翻译结果缓存（提升性能，降低API成本）

---

### 9. 后台管理服务（Admin Service）
github仓库地址： https://github.com/yeegeek/uyou-admin-service.git
**技术栈：** Go + PostgreSQL + Redis

**职责：**

- **员工系统：**
  - 员工登录/管理
  - 角色权限管理（RBAC）
  - 员工工资结算
- **推广系统：**
  - 邀请码生成与管理
  - 推广链接管理
  - 合作伙伴管理
  - 推广网站管理
  - 推广提成计算
- **客服系统：**
  - 用户投诉处理
  - 客服回复
  - 客服模板管理
- **内容审核：**
  - 图片审核队列
  - 审核记录管理
- **数据统计：**
  - 用户统计
  - 收入统计
  - 员工绩效统计
- **日志管理：**
  - API日志
  - 管理员操作日志
  - 错误日志

**定时任务：**

- 每日统计报表生成
- 更新合作伙伴提成比例
- 数据库优化
