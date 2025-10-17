# Robin-Camp 电影评分 API

基于 Kratos 框架实现的电影评分 API 服务。

## 项目概述

本项目实现了一个电影评分 API，支持：
- 电影信息管理（创建、查询、列表）
- 评分提交与聚合
- 票房数据集成
- RESTful API 接口

## 技术栈

- Go 1.23+
- Kratos v2 (微服务框架)
- PostgreSQL 16 (数据库)
- GORM v2 (ORM)
- Wire (依赖注入)
- Docker & Docker Compose

## 快速开始

### 前置要求

- Docker & Docker Compose
- Go 1.23+ (本地开发)
- Make

### 使用 Docker Compose 部署（推荐）

1. 复制环境变量配置：
```bash
cp .env.example .env
# 编辑 .env 文件，填入必要的配置
```

2. 启动所有服务：
```bash
make docker-up
```

3. 运行端到端测试：
```bash
make test-e2e
```

4. 停止服务：
```bash
make docker-down
```

### 本地开发

1. 安装 Kratos CLI：
```bash
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```

2. 安装依赖：
```bash
cd src
make init
```

3. 生成 Proto 代码：
```bash
make api
```

4. 生成 Wire 代码：
```bash
cd cmd/src
wire
```

5. 运行应用：
```bash
go run ./cmd/src -conf ./configs
```

## 项目结构

```
src/
├── api/                    # API 定义 (Protobuf)
│   └── movie/v1/          # Movie 服务 API
├── cmd/                   # 应用入口
│   └── src/
│       ├── main.go
│       ├── wire.go
│       └── wire_gen.go
├── configs/               # 配置文件
│   └── config.yaml
├── internal/              # 内部代码
│   ├── biz/              # 业务逻辑层
│   ├── data/             # 数据访问层
│   ├── service/          # 服务层
│   ├── server/           # 服务器配置
│   └── conf/             # 配置定义
└── third_party/          # 第三方 Proto 文件
```

## API 文档

详见项目根目录的 `openapi.yml` 文件。

主要端点：
- `POST /movies` - 创建电影（需认证）
- `GET /movies` - 查询电影列表
- `POST /movies/{title}/ratings` - 提交评分（需认证）
- `GET /movies/{title}/rating` - 获取聚合评分
- `GET /healthz` - 健康检查

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| PORT | 服务端口 | 8080 |
| DB_URL | PostgreSQL 连接字符串 | - |
| AUTH_TOKEN | API 认证 Token | - |
| BOXOFFICE_URL | 票房 API 地址 | - |
| BOXOFFICE_API_KEY | 票房 API Key | - |

## 开发命令

```bash
# 安装依赖
make init

# 生成 Proto 代码
make api

# 构建应用
make build

# 运行测试
make test

# Docker 构建
make docker-build

# 启动所有容器
make docker-up

# 停止所有容器
make docker-down

# 运行 E2E 测试
make test-e2e
```

## 设计文档

详细的设计方案请参考：
- `Design.md` - 架构设计文档
- `IMPLEMENTATION.md` - 实施细节文档

## 操作日志

- 初始化 Kratos 项目模板
- 更新数据库配置为 PostgreSQL
- 创建 Movie Service Proto 定义 (`api/movie/v1/movie.proto`)
- 设计数据库 Schema (`migrations/001_init_schema.sql`)
- 配置 Docker Compose 多服务编排
- 实现多阶段 Dockerfile（非 root 用户运行）
- 创建 Makefile 构建脚本
- 定义业务逻辑接口 (`internal/biz/types.go`)
- 实现电影创建与票房集成逻辑 (`internal/biz/movie.go`)
- 实现评分 Upsert 逻辑 (`internal/biz/rating.go`)
- 定义数据模型 (`internal/data/model.go`)
- 更新配置 Proto 支持票房 API 和认证 (`internal/conf/conf.proto`)
- 配置环境变量映射 (`configs/config.yaml`)

## License

MIT License
````

