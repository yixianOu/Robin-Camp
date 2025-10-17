# Robin-Camp 电影评分API实施文档

## 实施概览

本文档详细说明了基于 Kratos 框架实现电影评分 API 的完整方案。

## 已完成的设计与配置

### 1. 数据库设计
- ✅ 创建了 PostgreSQL 数据库迁移脚本 (`migrations/001_init_schema.sql`)
- ✅ 设计了 `movies` 表和 `ratings` 表
- ✅ 实现了必要的索引和约束
- ✅ 支持 Upsert 语义的唯一约束

### 2. 项目结构
- ✅ 遵循 Kratos 标准分层架构
- ✅ API 层：Proto 定义 (`api/movie/v1/movie.proto`)
- ✅ 业务层：UseCase 实现 (`internal/biz/`)
- ✅ 数据层：Repository 模式 (`internal/data/`)
- ✅ 服务层：HTTP/gRPC 转换 (`internal/service/`)

### 3. 配置管理
- ✅ 更新了配置 Proto (`internal/conf/conf.proto`)
- ✅ 配置文件模板 (`configs/config.yaml`)
- ✅ 环境变量支持
- ✅ Docker Compose 配置

### 4. 容器化
- ✅ 多阶段 Dockerfile（非 root 用户运行）
- ✅ Docker Compose 编排文件
- ✅ 数据库健康检查
- ✅ 自动迁移支持

### 5. 构建工具
- ✅ Makefile 包含所有必要命令
- ✅ docker-up、docker-down、test-e2e 目标

## 待实施的核心组件

### 1. Data 层实现

#### `internal/data/data.go` - 数据层初始化
```go
package data

import (
    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
    NewData,
    NewMovieRepo,
    NewRatingRepo,
    NewBoxOfficeClient,
)

// Data encapsulates database connection
type Data struct {
    db  *gorm.DB
    log *log.Helper
}

// NewData creates Data instance with database connection
func NewData(conf *conf.Data, logger log.Logger) (*Data, func(), error) {
    // Initialize PostgreSQL connection
    // Return Data instance and cleanup function
}
```

#### `internal/data/movie.go` - Movie Repository 实现
```go
package data

import (
    "context"
    "encoding/base64"
    "encoding/json"
    
    "Robin-Camp/internal/biz"
)

type movieRepo struct {
    data *Data
    log  *log.Helper
}

func NewMovieRepo(data *Data, logger log.Logger) biz.MovieRepo {
    return &movieRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

func (r *movieRepo) CreateMovie(ctx context.Context, movie *biz.Movie) error {
    // Convert biz.Movie to data.Movie
    // Execute GORM Create
}

func (r *movieRepo) ListMovies(ctx context.Context, query *biz.MovieListQuery) (*biz.MoviePage, error) {
    // Build dynamic query with filters
    // Implement cursor-based pagination
    // Return MoviePage with items and nextCursor
}
```

#### `internal/data/rating.go` - Rating Repository 实现
```go
package data

import (
    "context"
    
    "Robin-Camp/internal/biz"
    "gorm.io/gorm/clause"
)

type ratingRepo struct {
    data *Data
    log  *log.Helper
}

func NewRatingRepo(data *Data, logger log.Logger) biz.RatingRepo {
    return &ratingRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

func (r *ratingRepo) UpsertRating(ctx context.Context, rating *biz.Rating) (bool, error) {
    // Use GORM Clauses for ON CONFLICT DO UPDATE
    // clause.OnConflict{
    //     Columns:   []clause.Column{{Name: "movie_title"}, {Name: "rater_id"}},
    //     DoUpdates: clause.AssignmentColumns([]string{"rating", "updated_at"}),
    // }
}

func (r *ratingRepo) GetRatingAggregate(ctx context.Context, movieTitle string) (*biz.RatingAggregate, error) {
    // Execute SQL: SELECT ROUND(AVG(rating), 1), COUNT(*) FROM ratings WHERE movie_title = ?
}
```

#### `internal/data/boxoffice.go` - BoxOffice Client 实现
```go
package data

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "Robin-Camp/internal/biz"
    "Robin-Camp/internal/conf"
)

type boxOfficeClient struct {
    client  *http.Client
    baseURL string
    apiKey  string
    log     *log.Helper
}

func NewBoxOfficeClient(conf *conf.BoxOffice, logger log.Logger) biz.BoxOfficeClient {
    return &boxOfficeClient{
        client: &http.Client{
            Timeout: conf.Timeout.AsDuration(),
        },
        baseURL: conf.Url,
        apiKey:  conf.ApiKey,
        log:     log.NewHelper(logger),
    }
}

func (c *boxOfficeClient) GetBoxOffice(ctx context.Context, title string) (*biz.BoxOfficeData, error) {
    // Implement retry logic with exponential backoff
    // Call GET /boxoffice?title={title} with X-API-Key header
    // Parse response and return BoxOfficeData
    // Return nil on 404 or other errors (non-blocking)
}
```

### 2. Service 层实现

#### `internal/service/movie.go` - Service 实现
```go
package service

import (
    "context"
    
    v1 "Robin-Camp/api/movie/v1"
    "Robin-Camp/internal/biz"
)

type MovieService struct {
    v1.UnimplementedMovieServiceServer
    
    movieUC  *biz.MovieUseCase
    ratingUC *biz.RatingUseCase
}

func NewMovieService(movieUC *biz.MovieUseCase, ratingUC *biz.RatingUseCase) *MovieService {
    return &MovieService{
        movieUC:  movieUC,
        ratingUC: ratingUC,
    }
}

func (s *MovieService) CreateMovie(ctx context.Context, req *v1.CreateMovieRequest) (*v1.CreateMovieReply, error) {
    // Convert proto request to biz model
    // Call movieUC.CreateMovie
    // Convert biz model to proto response
    // Set Location header (handled in HTTP layer)
}

func (s *MovieService) SubmitRating(ctx context.Context, req *v1.SubmitRatingRequest) (*v1.SubmitRatingReply, error) {
    // Extract X-Rater-Id from context
    // Call ratingUC.SubmitRating
    // Return 201 if new, 200 if updated
}
```

### 3. Server 层实现

#### `internal/server/http.go` - HTTP 服务器配置
```go
package server

import (
    "context"
    
    v1 "Robin-Camp/api/movie/v1"
    "Robin-Camp/internal/conf"
    "Robin-Camp/internal/service"
    
    "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
    "github.com/go-kratos/kratos/v2/middleware/recovery"
    "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(c *conf.Server, authConf *conf.Auth, movieSvc *service.MovieService) *http.Server {
    var opts = []http.ServerOption{
        http.Middleware(
            recovery.Recovery(),
            AuthMiddleware(authConf.Token),
        ),
    }
    
    if c.Http.Network != "" {
        opts = append(opts, http.Network(c.Http.Network))
    }
    if c.Http.Addr != "" {
        opts = append(opts, http.Address(c.Http.Addr))
    }
    if c.Http.Timeout != nil {
        opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
    }
    
    srv := http.NewServer(opts...)
    v1.RegisterMovieServiceHTTPServer(srv, movieSvc)
    return srv
}

// AuthMiddleware - Bearer Token 验证
// RaterIdMiddleware - X-Rater-Id 提取
```

### 4. Wire 依赖注入

#### `cmd/src/wire.go`
```go
//go:build wireinject
// +build wireinject

package main

import (
    "Robin-Camp/internal/biz"
    "Robin-Camp/internal/conf"
    "Robin-Camp/internal/data"
    "Robin-Camp/internal/server"
    "Robin-Camp/internal/service"
    
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.BoxOffice, *conf.Auth, log.Logger) (*kratos.App, func(), error) {
    panic(wire.Build(
        server.ProviderSet,
        data.ProviderSet,
        biz.ProviderSet,
        service.ProviderSet,
        newApp,
    ))
}
```

### 5. 主程序

#### `cmd/src/main.go`
```go
package main

import (
    "flag"
    "os"
    
    "Robin-Camp/internal/conf"
    
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/config"
    "github.com/go-kratos/kratos/v2/config/file"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-kratos/kratos/v2/transport/http"
)

var (
    flagconf string
)

func init() {
    flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func main() {
    flag.Parse()
    logger := log.With(log.NewStdLogger(os.Stdout))
    
    // Load config from environment variables and file
    c := config.New(
        config.WithSource(
            file.NewSource(flagconf),
        ),
    )
    defer c.Close()
    
    if err := c.Load(); err != nil {
        panic(err)
    }
    
    var bc conf.Bootstrap
    if err := c.Scan(&bc); err != nil {
        panic(err)
    }
    
    // Override with environment variables
    overrideWithEnv(&bc)
    
    app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Boxoffice, bc.Auth, logger)
    if err != nil {
        panic(err)
    }
    defer cleanup()
    
    if err := app.Run(); err != nil {
        panic(err)
    }
}

func overrideWithEnv(bc *conf.Bootstrap) {
    // Override PORT, DB_URL, AUTH_TOKEN, BOXOFFICE_URL, BOXOFFICE_API_KEY
}
```

## 实施步骤

### Step 1: 安装依赖
```bash
cd src
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
go get -u github.com/go-kratos/kratos/v2
go get -u github.com/google/wire/cmd/wire
```

### Step 2: 生成 Proto 代码
```bash
cd src
make api
```

### Step 3: 实现核心代码
按照上述文档实现：
1. Data 层（Repository）
2. Biz 层（UseCase）- 已完成部分
3. Service 层
4. Server 层（中间件）
5. Wire 配置

### Step 4: 生成 Wire 代码
```bash
cd src/cmd/src
wire
```

### Step 5: 本地测试
```bash
# 启动数据库
docker compose up -d db

# 运行应用
cd src
go run ./cmd/src -conf ./configs

# 测试健康检查
curl http://localhost:8080/healthz
```

### Step 6: Docker 部署
```bash
# 构建并启动
make docker-up

# 运行 E2E 测试
make test-e2e

# 清理
make docker-down
```

## 关键技术要点

### 1. 分页游标实现
```go
type Cursor struct {
    Offset int `json:"offset"`
}

// 编码
func encodeCursor(offset int) string {
    data, _ := json.Marshal(&Cursor{Offset: offset})
    return base64.StdEncoding.EncodeToString(data)
}

// 解码
func decodeCursor(cursor string) (int, error) {
    data, err := base64.StdEncoding.DecodeString(cursor)
    var c Cursor
    json.Unmarshal(data, &c)
    return c.Offset, nil
}
```

### 2. 动态查询构建
```go
func (r *movieRepo) buildQuery(db *gorm.DB, q *biz.MovieListQuery) *gorm.DB {
    if q.Q != nil {
        db = db.Where("title ILIKE ?", "%"+*q.Q+"%")
    }
    if q.Year != nil {
        db = db.Where("EXTRACT(YEAR FROM release_date) = ?", *q.Year)
    }
    if q.Genre != nil {
        db = db.Where("LOWER(genre) = LOWER(?)", *q.Genre)
    }
    // ... 其他条件
    return db
}
```

### 3. BoxOffice 重试策略
```go
func (c *boxOfficeClient) GetBoxOffice(ctx context.Context, title string) (*biz.BoxOfficeData, error) {
    var lastErr error
    for i := 0; i < c.maxRetries; i++ {
        resp, err := c.doRequest(ctx, title)
        if err == nil {
            return resp, nil
        }
        lastErr = err
        time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // 指数退避
    }
    return nil, lastErr
}
```

### 4. 中间件实现
```go
func AuthMiddleware(token string) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            // 从 HTTP Header 提取 Authorization
            // 验证 Bearer Token
            // 仅对写操作生效（POST /movies, POST /ratings）
        }
    }
}

func RaterIdMiddleware() middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            // 从 HTTP Header 提取 X-Rater-Id
            // 注入到 context
        }
    }
}
```

## 验收检查清单

- [ ] 所有 API 端点符合 `openapi.yml` 规范
- [ ] 创建电影返回 201 状态码和 Location 头
- [ ] 评分 Upsert 语义正确（同 movie+rater 覆盖）
- [ ] 聚合评分保留 1 位小数
- [ ] 分页正确实现（cursor、limit、nextCursor）
- [ ] BoxOffice API 集成健壮（超时、重试、降级）
- [ ] 所有配置来自环境变量
- [ ] Docker 容器非 root 运行
- [ ] 健康检查端点可用
- [ ] E2E 测试全部通过
- [ ] 数据库迁移自动执行

## 下一步行动

1. **立即实施**：按照上述实施步骤，完成所有待实施组件
2. **代码生成**：运行 `make api` 和 `wire` 生成必要代码
3. **单元测试**：为每个层编写单元测试
4. **集成测试**：使用 testcontainers 进行集成测试
5. **文档完善**：编写 README.md（按要求不使用 AI）
6. **性能优化**：添加数据库连接池、缓存等优化

## 参考资料

- [Kratos 官方文档](https://go-kratos.dev/)
- [GORM 文档](https://gorm.io/docs/)
- [Wire 依赖注入](https://github.com/google/wire)
- [OpenAPI 规范](https://swagger.io/specification/)
