# 概述
实现可拆可合并的微服务。

# 系统架构

1. 一个discover服务器， 多个业务应用，业务应用通过discover发现其他业务应用 
2. 一个业务应用可以组合一个到多个模块
3. 同应用下的模块之间是直接调用， 而不同应用下的模块通过rpc调用

# 目录结构

```
g-micor/
├── application/                 # 应用启动层
│   ├── notifyApp/              # 通知应用
│   │   └── main.go             # 通知服务启动入口
│   └── xxxApp/                 # 用户应用
│       └── main.go             # 用户服务启动入口
├── contract/                   # 接口契约层
│   ├── notify_contract/        # 通知服务契约
│   │   ├── contract_gen.go     # 自动生成的契约代码
│   │   └── model.go            # 数据模型定义
│   └── xxx_contract/           # 用户服务契约
│       ├── contract_gen.go     # 自动生成的契约代码
│       └── model.go            # 数据模型定义
├── core/                       # 核心基础设施
│   ├── app/                    # 应用运行框架
│   │   └── run.go              # 应用启动逻辑
│   ├── discover/               # 服务发现
│   │   ├── client.go           # 服务发现客户端
│   │   ├── model.go            # 服务发现数据模型
│   │   └── server.go           # 服务发现服务端
│   ├── rpcx/                   # RPC通信框架
│   │   └── client.go           # RPC客户端
│   └── syncx/                  # 并发控制工具
├── service/                    # 业务服务层
│   ├── notify/                 # 通知模块
│   │   ├── email.go            # 邮件通知实现
│   │   └── package_gen.go      # 自动生成的包代码
│   ├── xx/                     # 业务模块
│   │   ├── package_gen.go      # 自动生成的包代码
│   │   └── xx.go               # 业务逻辑
│   └── gen.go                  # 服务生成工具  xx_gen.go的生成逻辑
├── go.mod                      # Go模块依赖管理
└── README.md                   # 项目说明文档
```

## 目录说明

- **application/**: 应用启动层，每个子目录代表一个独立的微服务应用，包含main.go作为启动入口
- **contract/**: 服务接口契约层，定义各服务之间的通信接口和数据模型
- **core/**: 核心基础设施层，提供服务发现、RPC通信、并发控制等基础能力
- **service/**: 业务服务实现层，包含具体的业务逻辑实现


# 开发步骤
1. 如果是全新应用 新建application, `app.Run("端口号", 服务发现地址,   服务模块.Service{})`
2. 新简模块 放于service目录下
3. 模块中直接编写方法，将需要暴露出去的方法，添加 // export 注释 ， 运行`service/gen.go` 生成服务对象和契约对象
4. 暴露方法规范： 
   1. 入参只有一个， 响应有结果和错误
   2. 入参和结果的结构体必须定义在契约目录下


protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative contract/notify/notify.proto