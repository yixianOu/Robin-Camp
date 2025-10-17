# 设计文档

## 技术栈
1. Go 1.25.1
2. Web Framework: Kratos v2
3. Database: PostgreSQL 16
4. ORM: GORM v2
5. 依赖注入: Wire
6. 协议: gRPC + HTTP (通过 Kratos 转码)

## 整体架构设计

### 1. 项目结构（Kratos 标准分层）
```
Robin-Camp/
├── api/                    # API定义层
│   └── movie/
│       └── v1/
│           ├── movie.proto          # Movie服务定义
│           ├── movie.pb.go          # Protobuf生成
│           ├── movie_http.pb.go     # HTTP绑定
│           └── movie_grpc.pb.go     # gRPC绑定
├── cmd/
│   └── server/
│       ├── main.go                  # 应用入口
│       ├── wire.go                  # Wire依赖注入配置
│       └── wire_gen.go              # Wire生成文件
├── internal/
│   ├── biz/                         # 业务逻辑层
│   │   ├── movie.go                 # 电影业务逻辑
│   │   ├── rating.go                # 评分业务逻辑
│   │   └── boxoffice.go             # 票房集成业务
│   ├── data/                        # 数据访问层
│   │   ├── data.go                  # Data层初始化
│   │   ├── movie.go                 # 电影Repository实现
│   │   ├── rating.go                # 评分Repository实现
│   │   └── model.go                 # GORM模型定义
│   ├── service/                     # 服务层（协议转换）
│   │   └── movie.go                 # Movie服务实现
│   ├── server/                      # Server配置
│   │   ├── http.go                  # HTTP服务器
│   │   └── grpc.go                  # gRPC服务器
│   └── conf/                        # 配置定义
│       ├── conf.proto               # 配置结构
│       └── conf.pb.go               # 生成的配置
├── configs/
│   └── config.yaml                  # 配置文件
├── migrations/                      # 数据库迁移
│   └── 001_init_schema.sql
├── Dockerfile                       # 多阶段构建
├── docker-compose.yml               # 容器编排
├── Makefile                         # 构建命令
└── .env.example                     # 环境变量示例
```

### 2. 数据库设计

#### 表结构

**movies 表**
```sql
CREATE TABLE movies (
    id VARCHAR(64) PRIMARY KEY,           -- 电影ID (m_uuid格式)
    title VARCHAR(255) NOT NULL,          -- 标题
    release_date DATE NOT NULL,           -- 上映日期
    genre VARCHAR(100) NOT NULL,          -- 类型
    distributor VARCHAR(255),             -- 发行商
    budget BIGINT,                        -- 预算(USD)
    mpa_rating VARCHAR(10),               -- MPA评级
    
    -- Box Office数据
    box_office_worldwide BIGINT,          -- 全球票房
    box_office_opening_usa BIGINT,        -- 美国首周票房
    box_office_currency VARCHAR(10),      -- 货币
    box_office_source VARCHAR(100),       -- 数据来源
    box_office_last_updated TIMESTAMP,    -- 最后更新时间
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_title (title),
    INDEX idx_year (EXTRACT(YEAR FROM release_date)),
    INDEX idx_genre (genre),
    INDEX idx_distributor (distributor),
    INDEX idx_budget (budget),
    INDEX idx_mpa_rating (mpa_rating)
);
```

**ratings 表**
```sql
CREATE TABLE ratings (
    id SERIAL PRIMARY KEY,
    movie_title VARCHAR(255) NOT NULL,    -- 电影标题（关联movies.title）
    rater_id VARCHAR(100) NOT NULL,       -- 评分者ID
    rating DECIMAL(2,1) NOT NULL          -- 评分值 (0.5-5.0)
        CHECK (rating >= 0.5 AND rating <= 5.0 AND MOD(rating * 10, 5) = 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (movie_title, rater_id),       -- Upsert约束
    INDEX idx_movie_title (movie_title),
    FOREIGN KEY (movie_title) REFERENCES movies(title) ON DELETE CASCADE
);
```

### 3. 核心业务流程

#### 3.1 电影创建流程 (POST /movies)
```
1. Service层接收HTTP请求
   ↓
2. 验证Bearer Token (中间件)
   ↓
3. Biz层：
   - 生成电影ID (m_<uuid>)
   - 创建电影基础记录
   ↓
4. 调用BoxOffice客户端查询票房数据
   - 成功(200): 合并数据（用户提供的优先）
   - 失败(非200): boxOffice设为null
   ↓
5. Data层持久化到数据库
   ↓
6. 返回201 + Location头
```

#### 3.2 评分提交流程 (POST /movies/{title}/ratings)
```
1. 验证X-Rater-Id头（中间件）
   ↓
2. Biz层：
   - 检查电影是否存在
   - Upsert评分（同movie_title+rater_id则更新）
   ↓
3. Data层：使用GORM的Upsert
   - ON CONFLICT (movie_title, rater_id) DO UPDATE
   ↓
4. 返回201(新建)或200(更新)
```

#### 3.3 评分聚合 (GET /movies/{title}/rating)
```
1. Data层执行SQL聚合查询:
   SELECT 
       ROUND(AVG(rating), 1) as average,
       COUNT(*) as count
   FROM ratings
   WHERE movie_title = ?
   ↓
2. 返回 {average, count}
```

#### 3.4 电影列表查询 (GET /movies)
```
1. 构建动态查询条件:
   - q: LIKE %keyword%
   - year: EXTRACT(YEAR FROM release_date) = ?
   - genre: genre = ? (case-insensitive)
   - distributor: distributor = ? (case-insensitive)
   - budget: budget <= ?
   - mpaRating: mpa_rating = ?
   ↓
2. 游标分页:
   - cursor解码获取offset
   - LIMIT limit+1 (检测是否有下一页)
   ↓
3. 返回 {items[], nextCursor}
```

### 4. 外部集成设计

#### BoxOffice API集成
```go
// internal/biz/boxoffice.go
type BoxOfficeClient interface {
    GetBoxOffice(ctx context.Context, title string) (*BoxOfficeData, error)
}

// 实现要点:
// - 超时控制: 3秒
// - 重试策略: 最多2次，指数退避
// - 降级处理: 错误不阻塞电影创建
// - 熔断: 连续失败5次后熔断30秒
```

### 5. 认证与鉴权

#### Bearer Token认证（写操作）
```go
// internal/server/http.go 中间件
func AuthMiddleware(token string) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            // 从Header提取Bearer Token
            // 验证是否等于配置的AUTH_TOKEN
        }
    }
}
```

#### X-Rater-Id认证（评分操作）
```go
func RaterIdMiddleware() middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            // 提取X-Rater-Id
            // 注入到context
        }
    }
}
```

### 6. 配置管理

#### 配置结构 (conf.proto)
```protobuf
message Bootstrap {
    Server server = 1;
    Data data = 2;
    BoxOffice boxoffice = 3;
    Auth auth = 4;
}

message Server {
    message HTTP {
        string network = 1;
        string addr = 2;
        google.protobuf.Duration timeout = 3;
    }
    HTTP http = 1;
}

message Data {
    message Database {
        string driver = 1;
        string source = 2;
    }
    Database database = 1;
}

message BoxOffice {
    string url = 1;
    string api_key = 2;
    google.protobuf.Duration timeout = 3;
}

message Auth {
    string token = 1;
}
```

#### 环境变量映射
```yaml
# configs/config.yaml
server:
  http:
    addr: 0.0.0.0:${PORT:8080}
    timeout: 30s
data:
  database:
    driver: postgres
    source: ${DB_URL}
boxoffice:
  url: ${BOXOFFICE_URL}
  api_key: ${BOXOFFICE_API_KEY}
  timeout: 3s
auth:
  token: ${AUTH_TOKEN}
```

### 7. 容器化部署

#### Dockerfile (多阶段构建)
```dockerfile
# Stage 1: Builder
FROM golang:1.25.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
WORKDIR /app
COPY --from=builder /server .
COPY configs/ ./configs/
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/server", "-conf", "/app/configs"]
```

#### docker-compose.yml
```yaml
version: '3.8'

services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: ${DB_NAME:-moviedb}
      POSTGRES_USER: ${DB_USER:-app}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-app}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app"]
      interval: 5s
      timeout: 5s
      retries: 5
    ports:
      - "5432:5432"

  app:
    build: ./src
    ports:
      - "${PORT:-8080}:8080"
    environment:
      PORT: ${PORT:-8080}
      DB_URL: postgres://${DB_USER:-app}:${DB_PASSWORD:-app}@db:5432/${DB_NAME:-moviedb}?sslmode=disable
      AUTH_TOKEN: ${AUTH_TOKEN}
      BOXOFFICE_URL: ${BOXOFFICE_URL}
      BOXOFFICE_API_KEY: ${BOXOFFICE_API_KEY}
    depends_on:
      db:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3

volumes:
  postgres_data:
```

### 8. API实现要点

#### Proto定义示例
```protobuf
// api/movie/v1/movie.proto
service MovieService {
    rpc CreateMovie(CreateMovieRequest) returns (CreateMovieReply);
    rpc ListMovies(ListMoviesRequest) returns (ListMoviesReply);
    rpc SubmitRating(SubmitRatingRequest) returns (SubmitRatingReply);
    rpc GetRating(GetRatingRequest) returns (GetRatingReply);
}

// HTTP绑定注解
rpc CreateMovie(CreateMovieRequest) returns (CreateMovieReply) {
    option (google.api.http) = {
        post: "/movies"
        body: "*"
    };
}
```

### 9. 数据库迁移

使用init容器或迁移工具（如golang-migrate）:
```sql
-- migrations/001_init_schema.sql
-- 包含上述CREATE TABLE语句
```

### 10. 关键技术点

1. **分页游标设计**: Base64编码的offset
2. **事务处理**: 电影创建+票房更新使用事务
3. **并发安全**: 评分Upsert使用数据库约束
4. **错误处理**: 统一错误码，符合OpenAPI规范
5. **日志**: 结构化日志（Kratos内置）
6. **监控**: Prometheus metrics（Kratos自带）

### 11. 测试策略

1. **单元测试**: 各层独立测试（mock依赖）
2. **集成测试**: 使用testcontainers
3. **E2E测试**: 使用提供的e2e-test.sh