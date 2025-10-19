# README.md
本项目的crud基本由ai vibe coding完成，因此本文主要介绍项目架构和技术亮点。

## 项目架构
1. 项目使用了kratos框架，在完成项目的需求的同时，学习和实践了kratos框架的使用方法和最佳实践。
2. 项目使用了最新版本的postgres和redis作为数据库和缓存，因此性能和扩展性较好。
3. 分层结构： 
   - api：通过proto文件定义，生成了服务的restfulAPI/grpc接口和对应的请求/响应结构体。
   - server：包含http/gprc服务器的启动和中间件配置。
   - service：负责对请求内容进行参数校验和调用业务逻辑。
   - biz：包含业务逻辑和服务实现，通过依赖倒置与data层解耦。
   - data：负责数据存储和访问，依赖biz层的接口。

## 技术亮点

### 响应处理：
1. customErrorEncoder：对于请求参数校验失败的情况，kratos默认返回400状态码，通过自定义错误编码器将其转换为422状态码，符合OpenAPI规范。
2. customResponseEncoder：对于创建电影接口，返回201 Created状态码，而不是默认的200 OK状态码。

### 鉴权中间件：
1. AuthMiddleware：判断请求Context的transport信息，如果是CreateMovie操作，则鉴权token
2. RaterIdMiddleware：判断请求Context的transport信息，如果是SubmitRating操作，则提取X-Rater-Id

### zset实现排行榜功能

### 自定义错误类型
1. 为电影提交评分、获取电影评分的时候，使用自定义error类型用以区分不同错误场景，如果目标电影不存在则返回404错误码，其他错误返回默认500错误码。

### 数据库操作
1. 使用uuid7作为电影主键，保证分布式环境下的唯一性和有序性。
2. 使用time.Now().UTC()与数据库交互，避免时区问题。在API响应中转换为本地时间，提升用户体验。
3. 使用GORM 的 ON CONFLICT 语法实现评分的插入或更新操作，简化代码逻辑。

### 单一源配置读取
1. .env中定义了部分配置，而kratos框架从config.yaml中读取配置，为了满足dont repeat yourself原则，config.yaml中的配置项使用环境变量占位符，而docker compose读取.env并在启动容器时注入环境变量，kratos解析到config.yaml中的环境变量占位符，就会读取环境变量作为配置值。

### 数据库索引

### 游标分页
1. ListMovies接口使用游标分页，接收游标作为offset，响应下一页的游标，用户只能逐页访问数据，避免了传统分页（大offset扫描）的性能问题。

## 未来可能的迭代
1. 使用validator中间件而不是在service层校验请求参数。使用customErrorEncoder丰富validator的错误响应（如http状态码）。
2. 目前更新数据库之后只是简单删除缓存，未来可以使用延时双删策略提升数据一致性。