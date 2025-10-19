# Robin-Camp 电影评分 API

基于 Kratos 框架实现的电影评分 API 服务。

## 项目概述

本项目实现了一个电影评分 API，支持：
- 电影信息管理（创建、查询、列表）
- 评分提交与聚合
- 票房数据集成
- RESTful API 接口

## 技术栈

- Go 1.25.1
- Kratos v2 (微服务框架)
- PostgreSQL 16 (数据库)
- GORM v2 (ORM)
- Wire (依赖注入)
- Docker & Docker Compose

## 快速开始

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

1. 安装依赖：
```bash
cd src
go mod tidy
```

2. 生成 Proto 代码：
```bash
cd src
make api
```

3. 生成 Wire 代码：
```bash
cd src/cmd/src
wire
```

4. 运行应用：
```bash
cd src
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

## Kratos DDD 架构说明

### 领域驱动设计 (DDD) 分层

Kratos 遵循严格的 DDD 分层架构，将代码按职责划分为四层：

#### 1. API 层 (`api/`)
**作用**：定义服务接口和数据契约
- 使用 Protobuf 定义服务 API（gRPC + HTTP）
- 通过 `google.api.http` 注解实现 gRPC-HTTP 转码
- 生成的代码包括：
  - `*.pb.go` - Protobuf 消息定义
  - `*_grpc.pb.go` - gRPC 服务端/客户端代码
  - `*_http.pb.go` - HTTP 服务端/客户端代码

**生成命令**：
```bash
# 安装 protoc 工具链
make init

# 生成 API 代码（api/ 目录下的 proto）
make api

# 生成配置代码（internal/conf/ 下的 proto）
make config
```

#### 2. Service 层 (`internal/service/`)
**作用**：协议转换和编排
- 实现 API 层定义的服务接口
- 负责 Protobuf ↔ 业务模型的转换
- 调用 Biz 层完成业务逻辑
- 不包含业务规则，仅做数据适配

**示例**：
```go
// service/movie.go
func (s *MovieService) CreateMovie(ctx context.Context, req *v1.CreateMovieRequest) (*v1.CreateMovieReply, error) {
    // 1. Proto → Biz 模型转换
    bizReq := convertProtoToBiz(req)
    
    // 2. 调用业务层
    movie, err := s.movieUC.CreateMovie(ctx, bizReq)
    
    // 3. Biz → Proto 模型转换
    return convertBizToProto(movie), nil
}
```

#### 3. Biz 层 (`internal/biz/`)
**作用**：核心业务逻辑（领域层）
- 包含业务规则、领域模型、用例（UseCase）
- 定义 Repository 接口（由 Data 层实现）
- 编排多个 Repository 完成复杂业务流程
- 不依赖具体的数据库或外部服务实现

**关键概念**：
- **领域模型**：业务实体（如 `Movie`, `Rating`）
- **UseCase**：业务用例（如 `MovieUseCase`, `RatingUseCase`）
- **Repository 接口**：数据访问抽象（如 `MovieRepo`, `RatingRepo`）

**示例**：
```go
// biz/movie.go
type MovieUseCase struct {
    repo            MovieRepo              // 依赖接口，非实现
    boxOfficeClient BoxOfficeClient
}

func (uc *MovieUseCase) CreateMovie(ctx context.Context, req *CreateMovieRequest) (*Movie, error) {
    // 1. 生成业务 ID
    movie := &Movie{ID: "m_" + uuid.New().String(), ...}
    
    // 2. 调用外部服务（票房数据）
    boxOffice, _ := uc.boxOfficeClient.GetBoxOffice(ctx, req.Title)
    
    // 3. 业务规则：合并数据（用户提供优先）
    mergeBoxOfficeData(movie, boxOffice)
    
    // 4. 持久化
    return uc.repo.CreateMovie(ctx, movie)
}
```

#### 4. Data 层 (`internal/data/`)
**作用**：数据访问实现（基础设施层）
- 实现 Biz 层定义的 Repository 接口
- 管理数据库连接、缓存、外部 API 调用
- 处理 GORM 模型 ↔ 领域模型转换
- 实现缓存策略（Redis）、事务管理

**组件**：
- `data.go` - 初始化数据库/Redis 连接，提供 `*Data` 结构
- `model.go` - GORM 数据模型（对应数据库表）
- `movie.go` - `MovieRepo` 接口实现（含 Redis 缓存）
- `rating.go` - `RatingRepo` 接口实现（含 Redis ZSet 排行榜）
- `boxoffice.go` - 外部 HTTP 客户端实现

**示例**：
```go
// data/movie.go
type movieRepo struct {
    data *Data  // 包含 db *gorm.DB 和 rdb *redis.Client
}

func (r *movieRepo) CreateMovie(ctx context.Context, movie *biz.Movie) error {
    // 1. 领域模型 → GORM 模型
    m := bizToModel(movie)
    
    // 2. 数据库操作
    if err := r.data.db.Create(&m).Error; err != nil {
        return err
    }
    
    // 3. 更新缓存
    r.data.rdb.Set(ctx, "movie:"+movie.Title, json, 15*time.Minute)
    
    return nil
}
```

#### 5. Server 层 (`internal/server/`)
**作用**：服务器配置和中间件
- 初始化 HTTP/gRPC 服务器
- 注册服务路由
- 配置中间件（认证、日志、恢复）

**生成命令**：
```bash
# 创建新的 HTTP/gRPC 服务器配置（初始化项目时）
kratos new <project-name>
```

#### 6. Conf 层 (`internal/conf/`)
**作用**：配置结构定义
- 使用 Protobuf 定义配置结构
- 通过 Kratos 配置加载器读取 YAML/JSON
- 支持环境变量替换

**生成命令**：
```bash
make config  # 生成 conf.pb.go
```

### Wire 依赖注入

Kratos 使用 [Wire](https://github.com/google/wire) 实现编译时依赖注入：

**配置文件** (`cmd/src/wire.go`):
```go
//go:build wireinject
// +build wireinject

func wireApp(*conf.Server, *conf.Data, *conf.BoxOffice, *conf.Auth, log.Logger) (*kratos.App, func(), error) {
    panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
```

**生成命令**：
```bash
# 在 cmd/src/ 目录执行
go generate ./...
# 或直接运行
wire
```

**生成文件**：`wire_gen.go` - 包含完整的依赖注入代码

### Kratos CLI 命令总结

```bash
# 1. 创建新项目（生成标准目录结构）
kratos new <project-name>

# 2. 生成 Proto Service（生成 api/ 下的模板）
kratos proto add api/<service>/<version>/<service>.proto

# 3. 生成 Service 实现（根据 proto 生成 service 层代码）
kratos proto client api/<service>/<version>/<service>.proto

# 4. 生成 Server 代码（生成 internal/server/*.go）
kratos proto server api/<service>/<version>/<service>.proto -t internal/service

# 本项目实际使用的命令
make init     # 安装 protoc 工具链
make api      # 生成 api/movie/v1/*.pb.go
make config   # 生成 internal/conf/conf.pb.go
go generate   # 生成 wire_gen.go
```

### 数据流向示例

**创建电影请求流程**：
```
HTTP Request (POST /movies)
    ↓
HTTP Server (internal/server/http.go) + 认证中间件
    ↓
MovieService.CreateMovie (internal/service/movie.go)
    - Proto → Biz 模型转换
    ↓
MovieUseCase.CreateMovie (internal/biz/movie.go)
    - 生成业务 ID
    - 调用 BoxOfficeClient 获取票房数据
    - 业务规则：合并数据
    ↓
MovieRepo.CreateMovie (internal/data/movie.go)
    - Biz → GORM 模型转换
    - 写入 PostgreSQL
    - 更新 Redis 缓存
    ↓
返回结果 (201 + Location header)
```

### 关键设计原则

1. **依赖倒置**：Biz 层定义接口，Data 层实现接口
2. **单向依赖**：外层依赖内层（Service → Biz → Data），反向通过接口
3. **领域隔离**：Biz 层使用纯业务模型，不依赖 ORM 或 Proto
4. **协议无关**：Biz 层不知道上层是 HTTP 还是 gRPC
5. **可测试性**：每层都可以通过 Mock 接口独立测试

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

## Makefile 命令

项目提供以下 Makefile 命令：

```bash
# 构建并启动全部容器（包含数据库和应用）
make docker-up

# 停止并清理所有容器
make docker-down

# 运行端到端测试
make test-e2e
```

## 常用开发命令

```bash
# 安装依赖
cd src && go mod tidy

# 生成 Proto 代码
cd src && make api

# 生成 Wire 依赖注入代码
cd src && go generate ./...

# 构建应用
cd src && go build -o ../bin/server ./cmd/src

# 运行应用
cd src && go run ./cmd/src -conf ./configs

# 运行单元测试
cd src && go test -v ./...
```

## 设计文档

详细的设计方案请参考：
- `Design.md` - 架构设计文档
- `IMPLEMENTATION.md` - 实施细节文档

### 🎯 完成的功能
1. **电影管理**：创建、查询、列表、搜索、分页
2. **评分系统**：提交评分（Upsert 语义）、聚合计算、Redis 缓存
3. **票房集成**：异步调用上游 API，失败不阻塞创建流程
4. **认证授权**：Bearer Token（创建电影）、X-Rater-Id（提交评分）
5. **错误处理**：统一错误码（401/404/422）、自定义 CODEC 错误处理
6. **数据持久化**：PostgreSQL + GORM 软删除、Redis 排行榜
7. **API 契约**：Proto3 定义、HTTP/gRPC 双协议、OpenAPI 兼容

### 📊 架构质量
- ✅ DDD 四层架构（API → Service → Biz → Data）
- ✅ 依赖注入（Wire 自动生成）
- ✅ 中间件系统（认证、恢复、日志）
- ✅ 缓存策略（Redis 电影缓存、ZSet 排行榜）
- ✅ 数据库索引优化（标题、年份、类型、预算）
- ✅ Docker Compose 部署（多服务编排、健康检查）

## 操作日志

本节记录项目开发过程中执行的关键操作,便于回溯与复现。

### 2025-01-19：参数校验分层重构

**问题**：参数校验逻辑放在了 Biz 层,违反了 DDD 分层职责原则。

**DDD 分层职责**：
- Service 层：协议转换、**参数校验**、调用 Biz 层
- Biz 层：业务逻辑、业务规则、领域模型

**参数校验 vs 业务规则**：
- 参数校验：检查输入格式是否合法（如：评分 0.5-5.0，步长 0.5）→ Service 层
- 业务规则：检查业务约束（如：同一用户一天只能评分一次）→ Biz 层

**重构步骤**：
```bash
# 1. 将 isValidRating() 函数从 Biz 层移到 Service 层
# 编辑 src/internal/service/movie.go，添加参数校验

# 2. 在 Service 层的 SubmitRating 方法中添加校验
# 校验失败直接返回 422，不经过 Biz 层

# 3. 移除 Biz 层的参数校验逻辑
# 编辑 src/internal/biz/rating.go：
# - 删除 ErrInvalidRating 错误定义
# - 删除 isValidRating() 函数
# - 删除 SubmitRating 中的参数校验代码

# 4. 移除 Service 层对 ErrInvalidRating 的处理

# 5. 验证
cd src && go build ./...
cd .. && bash ./e2e-test.sh
# 结果：28/28 tests passed ✅
```

**重构收益**：
- ✅ 符合 DDD 分层职责原则
- ✅ Service 层统一处理所有参数校验
- ✅ Biz 层专注于业务逻辑
- ✅ 代码结构更清晰

### 2025-01-19：错误处理重构

1. **发现代码异味**
   - Service 层使用硬编码字符串判断错误类型（`err.Error() == "..."`）
   - 请求结束后设置无效的 Context 值（死代码）

2. **重构步骤**
   ```bash
   # 1. 在 Biz 层定义 sentinel errors
   # 编辑 src/internal/biz/rating.go，添加：
   # var (
   #     ErrMovieNotFound = errors.New("movie not found")
   # )
   
   # 2. 修改 Biz 层错误包装
   # 使用 fmt.Errorf("%w: ...", ErrMovieNotFound, err) 保留错误链
   
   # 3. 修改 Service 层错误检查
   # 编辑 src/internal/service/movie.go：
   # - 移除硬编码字符串判断
   # - 使用 errors.Is(err, biz.ErrMovieNotFound) 类型安全检查
   # - 删除死代码：ctx = context.WithValue(ctx, "is_new_rating", isNew)
   
   # 4. 验证编译
   cd src && go build ./...
   
   # 5. 运行 E2E 测试验证
   cd .. && bash ./e2e-test.sh
   # 结果：28/28 tests passed ✅
   ```

3. **重构收益**
   - ✅ 类型安全：编译期检查错误类型
   - ✅ 可维护性：错误定义单点维护
   - ✅ 性能提升：指针比较 vs 字符串比较
   - ✅ 代码清晰：删除死代码和误导性注释
   - ✅ 符合 Go 1.13+ 错误处理最佳实践

## License

MIT License
````

