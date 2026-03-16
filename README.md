
---

## 📖 模块说明（详细）

### 🔹 `cmd/` 项目启动入口目录

- `cmd/appconfig/appconfig.go`  
  项目的配置文件的映射结构体定义。

- `cmd/global/global.go`  
  存放全局变量，如需使用项目中任何地方可访问的变量，可定义在此处。

- `cmd/initialize/init.go`  
  项目的初始化逻辑所在，包括加载配置、初始化数据库连接、定时任务注册等启动前的前置操作。

- `cmd/appconfig.yaml`  
  配置文件，包括 nacos 配置信息、项目端口等服务运行所需配置项。

- `cmd/main.go`  
  项目主入口函数，程序从此处启动。

---

### 🔹 `core/` 项目核心目录

- `core/db/`  
  数据库操作层。细化为：
    - `mongodb_methods/`：MongoDB 数据操作方法，建议按照业务模块拆分（如 order、goods）。
    - `mysql_methods/`：MySQL 数据操作方法，也应按照模块分类组织。

- `core/handle/`  
  逻辑处理层。如果项目是 serve 服务类型，业务逻辑处理可集中写在这里，保持控制器轻量化。

- `core/http_method/`  
  外部接口封装层。如 Go 项目需调用 Node 接口、或其他微服务接口，应将调用逻辑封装于此。建议以模块方式划分（如 order、goods 等）。

- `core/scheduler/`  
  定时任务模块，适用于需要周期性处理任务的服务。包含：
    - `consumer/`：定时任务消费者逻辑（即任务消费处理部分），也建议按照模块拆分。
    - `jobs/`：定时任务生产者逻辑，若使用队列则为生产者；若无队列可直接写业务逻辑。
    - `register_tasks.go`：注册定时任务的核心入口文件，用于将任务添加到调度器中。

- `core/tools/`  
  项目工具类集合，如通用方法封装、日志工具、加密工具等。
    - `aes.go`：AES 加解密工具。
    - `zap.go`：基于 zap 的日志封装，便于统一日志格式输出。

---

## 项目结构说明

```text

github.com/sbigtree/go-package-service/
├── cmd/ # 项目入口目录
│ ├── appconf/ # 项目配置结构体定义（用于映射配置文件）
│ │ └── appconfig.go
│ ├── global/ # 项目全局变量统一管理
│ │ └── global.go
│ ├── initialize/ # 项目初始化逻辑（配置、DB、定时任务等）
│ │ └── init.go
│ ├── appconfig.yaml # 项目配置文件（含 nacos 配置、端口等）
│ └── main.go # 项目启动入口
│
├── core/ # 项目核心目录
│ ├── db/ # 数据库操作层
│ │ ├── mongodb_methods/ # MongoDB 操作方法（按模块拆分）
│ │ └── mysql_methods/ # MySQL 操作方法（按模块拆分）
│ ├── handle/ # 逻辑处理层（serve 服务业务逻辑入口）
│ ├── http_method/ # 外部 HTTP 接口调用封装（如调用 Node 接口）
│ ├── scheduler/ # 定时任务相关逻辑
│ │ ├── consumer/ # 定时任务消费者处理逻辑（按模块拆分）
│ │ ├── jobs/ # 定时任务/队列生产者逻辑实现
│ │ └── register_tasks.go # 注册定时任务的入口文件
│ └── tools/ # 工具包目录（日志、加解密等）
│   ├──aes.go # AES 加解密工具
│   └──zap.go # Zap 日志封装
│
├── logs/ # 项目日志文件存放目录（按日期分）
├── .gitignore # Git 忽略文件配置
├── go.mod # Go Modules 管理文件
└── README.md # 项目说明文档

```

## 🚀 快速启动

```bash
# 1. 安装依赖
go mod tidy

# 2. 启动服务
go run cmd/main.go

```
```
#  作为serve服务可参考go-goods-serve的使用

#  作为定时任务可参考go-task-serve的使用
```
