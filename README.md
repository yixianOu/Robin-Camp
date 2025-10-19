# README.md
本项目的curl基本由ai vibe coding完成，因此本文主要介绍项目架构和技术亮点。

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
2. customResponseEncoder：

### 鉴权中间件：
1. AuthMiddleware：判断请求Context的transport信息，如果是CreateMovie操作，则鉴权token
2. RaterIdMiddleware：判断请求Context的transport信息，如果是SubmitRating操作，则提取X-Rater-Id

### zset实现排行榜功能

### SubmitRating功能实现

### 未来可能的迭代
1. 使用validator中间件而不是在service层校验请求参数。使用customErrorEncoder丰富validator的错误响应（如http状态码）。