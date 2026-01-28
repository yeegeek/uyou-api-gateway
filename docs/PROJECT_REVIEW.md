# 项目审查报告 - SNS社交网络服务API

## 一、项目概述

这是一个基于 **PHP** 的跨国交友平台后端API项目，采用 **Slim 3.5** 微框架构建，使用 **Laravel Eloquent 5.3** 作为ORM。项目支持多平台入口（SNS用户端、员工管理端、管理员后台）。

---

## 二、功能模块总结

### 1. 用户认证与授权模块
- 用户注册/登录（支持JWT Token认证）
- 第三方登录（Facebook、微信）
- 密码找回、快速登录、Token刷新

### 2. 用户管理模块
- 用户资料管理、搜索与推荐
- 在线用户管理、生日用户
- 用户评价系统、VIP会员管理

### 3. 社交功能模块
- **好友系统**：好友请求/接受/删除、好友分组、黑名单、备注
- **消息系统**：私信发送/接收、对话线程管理、消息翻译、草稿保存、消息撤回、礼物发送
- **动态系统**：发布/删除/查看动态
- **互动功能**：点赞、访客记录

### 4. 内容管理模块
- 图片/视频上传（支持收费图片）
- 内容审核、文件审核
- 人脸识别检测

### 5. 支付系统模块
- 充值、VIP升级、付费图片购买
- 交易记录、合作伙伴提现

### 6. 翻译服务模块
- 机器翻译（Google Translate）

### 7. 员工/管理员系统
- 员工登录/管理
- 工资结算、推广管理
- 档案申请/绑定

### 8. 推广系统模块
- 邀请码生成、推广链接管理
- 合作伙伴管理、SMTP邮件配置

### 9. 商品与订单模块
- 商品管理、订单管理

### 10. 视频通话模块
- 视频通话创建/状态更新/结束处理

### 11. 通知系统模块
- 实时推送、离线推送、通知历史

### 12. 搜索模块
- 用户搜索、地理位置搜索
- Elasticsearch全文搜索

### 13. AI功能模块
- 消息建议、用户分析、AI检测

### 14. 客服支持模块
- 用户投诉、客服回复、模板管理

---

## 三、高级技术特性

### 1. 队列系统（基于Redis）

队列控制器位于 `src/Console/QueueController.php`，使用 Redis List 实现异步任务处理。

**队列任务类型**（位于 `src/Console/Commands/`）：
- `WriteApiLog` - API日志写入
- `WriteLog` - 调试日志写入
- `SendEmail` - 发送邮件
- `SendSms` - 发送短信
- `SendNotification` - 发送推送通知
- `ProcessEmail` - 处理邮件

### 2. 定时任务系统

定时任务控制器位于 `src/Console/CronController.php`，配置文件位于 `src/Config/cron.php`。

**每分钟执行的任务**：
- `processEmail` - 发送邮件
- `deleteInvalidCodes` - 删除过期验证码
- `harassUser` - 用户联系操作
- `machineTranslate` - 机器翻译
- `finishCall` - 处理通话断线

**每小时执行的任务**：
- `createProfile` - 创建档案
- `getExchangeRate` - 获取汇率
- `generateOnlineUser` - 生成在线用户
- `updateFingerprint` - 更新登录数据
- `deliverOrder` - 礼物自动发货
- `addFace` - 人脸识别上传

**定时任务**（特定时间）：
- `16:00` - `dailyRoutine` 每日例行操作
- `16:01` - `generateTodayBirthday` 今日生日用户
- `16:02` - `generateLatestUser` 最新用户
- `08:00` - `emailReminder` 邮件提醒
- `09:00` - `updateAge`, `updateInactive` 更新用户年龄和状态

### 3. 缓存系统（Redis）

缓存适配器位于 `src/Cache/RedisAdapter.php`，使用 Predis 客户端。

支持的操作：
- 键值存储（get/put/forever）
- 列表操作（append/prepend/lists）
- 队列操作（first/last）
- 原子自增

### 4. 数据库读写分离

配置位于 `src/Config/db.php`，支持三个数据库连接：
- `default` - 默认连接（读写分离）
- `write` - 写入连接
- `email` - 邮件数据库连接

---

## 四、第三方服务集成

### AWS服务

| 服务 | 用途 |
|------|------|
| **S3** | 文件存储（资源、公共、私密三个存储桶） |
| **CloudFront** | CDN加速 |
| **Lambda** | 图片处理 |
| **Rekognition** | 人脸识别（检测、搜索、敏感内容检测、名人识别） |
| **Aurora RDS** | 数据库 |
| **ElastiCache** | Redis缓存 |
| **SNS** | 消息通知 |
| **SES** | 备用邮件服务 |

### 通信服务

| 服务 | 用途 |
|------|------|
| **Mailgun** | 邮件服务（路由、webhooks、退回处理） |
| **云片** | 短信服务 |
| **Pusher/WebSocket** | 实时通信 |
| **Firebase** | 移动端推送通知 |

### 支付服务

| 服务 | 用途 |
|------|------|
| **PayPal** | 国际支付 |
| **微信支付** | 国内支付 |

### 认证与验证

| 服务 | 用途 |
|------|------|
| **Facebook OAuth** | 第三方登录 |
| **微信开放平台** | 微信登录 |
| **Google reCAPTCHA** | 验证码 |
| **Geetest** | 验证码服务 |

### 其他服务

| 服务 | 用途 |
|------|------|
| **Google Translate API** | 机器翻译 |
| **OpenAI GPT** | AI消息建议/用户分析 |
| **SerpAPI** | Google搜索结果 |
| **GeoIP2** | IP地理位置 |
| **Elasticsearch** | 全文搜索 |
| **Have I Been Pwned** | 邮箱泄露检测 |

---

## 五、中间件列表

| 中间件 | 功能 |
|--------|------|
| `Auth` | JWT认证 |
| `Pagination` | 分页处理 |
| `AccessControl` | 权限控制 |
| `TokenToUser` | Token转用户 |
| `SetHostname` | 设置主机名 |
| `SetUserLanguage` | 设置用户语言 |
| `SetClientInfo` | 设置客户端信息 |
| `Validations` | 输入验证 |
| `IpRateLimit` | IP限流 |
| `Maintainance` | 维护模式 |
| `Benchmark` | 性能监控 |
| `CacheControl` | 缓存控制 |
| `HandleHeaders` | 响应头处理 |
| `RecordApi` | API记录 |

---

## 六、项目统计

| 类型 | 数量 |
|------|------|
| 控制器 | 46个 |
| 数据模型 | 148个 |
| 中间件 | 15个 |
| 命令行任务 | 60+个 |
| Traits | 19个 |
| 邮件模板 | 19个 |
| 支持语言 | 8种（中英日韩法德意繁体） |

---

## 七、安全特性

- JWT Token认证
- IP限流（`IpRateLimit`中间件）
- 输入验证（`respect/validation`）
- 密码加密
- 内容审核（AI）
- 黑名单机制
- 敏感内容检测
- 邮箱泄露检测（Have I Been Pwned）

---

## 八、项目结构

```
api/
├── src/                    # 源代码目录
│   ├── bootstrap.php       # 应用启动文件
│   ├── init.php            # 初始化文件
│   ├── settings.php        # 应用配置
│   ├── dependencies.php    # 依赖注入配置
│   ├── middleware.php      # 中间件配置
│   ├── Controllers/        # 控制器 (46个)
│   ├── Models/             # 数据模型 (148个)
│   ├── Middleware/         # 中间件 (15个)
│   ├── Routes/             # 路由定义
│   │   ├── sns.php         # SNS用户端路由
│   │   ├── admin.php       # 管理端路由
│   │   ├── common.php      # 公共路由
│   │   └── command.php     # 命令行路由
│   ├── Console/            # 命令行工具
│   │   ├── Commands/       # 命令类 (60+个)
│   │   └── Kernel.php      # 命令调度器
│   ├── Dependencies/       # 依赖配置
│   ├── Config/             # 配置文件
│   ├── Libs/               # 工具库
│   ├── Storage/            # 存储适配器
│   ├── Cache/              # 缓存适配器
│   ├── Email/              # 邮件模板
│   ├── Lang/               # 多语言文件
│   ├── Rekognition/        # 人脸识别
│   └── Traits/             # 公共Traits
├── public/                 # Web入口
│   ├── sns/                # SNS入口
│   ├── admin/              # 管理后台入口
│   └── redirect/           # 重定向入口
├── bin/                    # 脚本文件
├── tests/                  # 测试文件
├── doc/                    # 文档
├── deployment/             # 部署配置
└── composer.json           # 依赖管理
```

---

## 九、主要依赖包

```json
{
  "slim/slim": "^3.5",
  "illuminate/database": "^5.3",
  "predis/predis": "^1.1",
  "aws/aws-sdk-php": "^3.19",
  "elasticsearch/elasticsearch": "^6.0",
  "mailgun/mailgun-php": "^2.1",
  "pusher/pusher-php-server": "^2.6",
  "omnipay/paypal": "^2.6",
  "lokielse/omnipay-wechatpay": "^1.0",
  "facebook/graph-sdk": "^5.5",
  "kreait/firebase-php": "^4.35",
  "tuupola/slim-jwt-auth": "^2.3",
  "robmorgan/phinx": "^0.8.0",
  "geoip2/geoip2": "~2.0"
}
```

---

*报告生成时间：2026年1月*
