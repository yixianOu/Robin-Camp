# 电影评分 API — 本地实现（Take‑home）

> 目标：在本地实现一个 **Movie Rating API** 服务，满足给定的 `openapi.yml` 合同，并在新增电影时调用提供的 **票房（Box Office）Mock API** 进行数据整合。我们只验证本地运行与端到端测试，无需任何云厂商账号。

---

## 你将拿到的文件

* `openapi.yml`：主服务 API 合同（必须严格遵守）。
* `boxoffice.openapi.yml`：票房三方接口的合同说明。
* `e2e-test.sh`：端到端测试脚本（使用 `curl` 与 `jq`）。

> 端到端脚本默认访问 `http://127.0.0.1:8080`，并从环境变量读取鉴权 Token；如脚本内有额外说明，以脚本为准。

---

## 你的交付物

1. **服务实现源码**（建议 Go）：完全按 `openapi.yml` 行为返回；读公开，写需鉴权。
2. **Dockerfile**：多阶段构建，最终镜像非 root 运行，监听 `0.0.0.0:8080`。
3. **docker-compose.yml**：一条命令拉起 **应用服务** 与 **数据库**。数据库容器需健康检查。
4. **数据库设计与迁移**：
   * 需要设计完整的数据库 schema，包含电影表、评分表等必要表结构
   * 提供数据库迁移文件
   * 在 `docker compose up` 后自动执行，创建所需的表结构
5. **健康检查**：提供 `GET /healthz` 返回 200。
6. **Make 目标或脚本**：

   * `make docker-up`：构建并启动全部容器，等待健康检查通过。
   * `make docker-down`：停止并清理。
   * `make test-e2e`：在服务就绪后执行 `./e2e-test.sh`。
7. 更新 **.env.example**：列出所需环境变量。
8. **README.md** 详细描述设计思路：
   1. 数据库选型 和 设计
   2. 后端服务选型 和 设计
   3. 在完成项目后，思考一下整体项目还有哪些可以优化的内容
   4. **注意千万不要用AI写这个文件，这非常容易识别，假如识别出来，会直接判定失败**

---

## 运行要求（本地）

* 必备：`docker`、`docker compose`、`bash`、`curl`、`jq`。
* 服务默认监听：`127.0.0.1:8080`（端口可通过 `PORT` 覆盖）。
* **不得**依赖任何云服务；票房查询仅调用提供的 Apifox Mock URL。

---

## 环境变量（固定命名）

* `PORT`：服务端口，默认 `8080`。
* `AUTH_TOKEN`：写操作所需的静态 Bearer Token。
* `DB_URL`：应用连接字符串（在 Compose 内通常形如 `postgres://app:app@db:5432/app?sslmode=disable`）。
* `BOXOFFICE_URL`：Apifox Mock 基础 URL（由我们在任务包中告知）。
* `BOXOFFICE_API_KEY`：调用 Apifox 所需的静态 Key。

> 评测将检查：配置是否全部来源于环境变量，严禁在代码中硬编码密钥或 URL。

---

## 功能与行为要点（概述）

* **电影创建** `POST /movies`：
  * 成功创建后，同步调用票房接口 `GET /boxoffice?title=...`。
  * 若上游返回 **200**：将 `{gross_usd, currency, source, last_updated}` 合并进电影记录。
  * 若不成功， 比如 **404**：`box_office = null`， 不阻塞创建。

* **评分上报** `POST /movies/{title}/ratings`（需鉴权，要求请求头 `X-Rater-Id`）：
  * **Upsert** 语义：同 `(movie_title, rater_id)` 的评分再次提交会覆盖。
  * `rating` 取值集合：`{0.5, 1.0, …, 5.0}`。

* **评分聚合** `GET /movies/{title}/rating`：返回 `{average, count}`，其中平均值四舍五入保留 **1 位小数**。

* **列表与检索** `GET /movies`：支持 `q | year | genre | limit | cursor`，分页合同稳定（`items[]` + `next_cursor`）。

* **错误模型与状态码**：严格按 `openapi.yml`（创建返回 **201**，并带 `Location` 头；常见错误 400/401/403/404）。

> 以上仅为摘要；以 `openapi.yml` 为最终合同。

---

## 本地运行与测试（建议流程）

1. 复制并填写环境：

   ```bash
   cp .env.example .env
   # 填好 AUTH_TOKEN, BOXOFFICE_URL, BOXOFFICE_API_KEY
   ```

2. 一键启动：

   ```bash
   make docker-up
   # 或者：docker compose up -d --build
   ```

3. 端到端验证：

   ```bash
   make test-e2e
   # 等价：./e2e-test.sh
   ```

4. 清理：

   ```bash
   make docker-down
   ```

---

## 验收标准

* **合同一致**：路径、请求/响应体、错误模型、分页字段与状态码全部符合 `openapi.yml`；创建返回 **201** 且含 `Location` 头。
* **评分幂等**：`POST /ratings` 为 Upsert；聚合值与计数正确，平均值保留 1 位小数。
* **三方集成健壮**：遵循超时/重试/降级策略；上游出错不影响创建成功。
* **Docker 一键可用**：`docker compose up` 即可启动服务与数据库，迁移自动执行；日志输出到标准输出；`/healthz` 可探活。
* **配置合规**：仅使用环境变量注入配置；无密钥/URL 硬编码。

---

## 提交方式

* 提交 Git 仓库链接或压缩包（含源码、Dockerfile、docker-compose.yml、数据库迁移文件、Makefile、README、.env.example）。
* 请附上可复现的提交哈希（commit）或版本号。

祝完成顺利。以质量为先，避免过度工程化；先让 E2E 全绿，再做小步改进。
