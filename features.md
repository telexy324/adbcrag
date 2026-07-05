# 运维知识库 RAG 系统开发说明

## 1. 项目目标

开发一个面向运维场景的知识库问答系统。

系统基于内部运维文档、操作手册、告警处置手册、应急预案、变更回滚方案等资料，构建 RAG 知识库。

用户可以上传文档，并基于知识库提问。系统需要先检索知识库内容，再调用内网 DeepSeek v4 LLM 模型生成答案。

系统还需要支持对接 Elasticsearch 和服务器内指定路径的日志文件，结合已发布运维文档和 LLM 接口做日志检索、异常摘要、根因分析和处置建议。

本项目只用于辅助运维分析，不自动执行任何生产命令。

---

## 2. 技术栈

### 前端

* React
* TypeScript
* Vite
* shadcn/ui
* Tailwind CSS
* React Router
* TanStack Query
* Axios

### 后端

* Golang
* Gin Web Framework
* GORM
* PostgreSQL
* pg_trgm
* MinIO 或本地文件存储
* Elasticsearch REST API
* SSH / SFTP
* 内网 DeepSeek v4 LLM 模型

### 检索策略

当前环境只有 DeepSeek v4 LLM，没有独立 embedding 模型。

因此第一阶段不使用 pgvector / embedding，改用以下方案尽量接近 RAG 效果：

1. 文档切片后，调用 DeepSeek 为每个 chunk 生成检索增强信息：
   - 摘要 summary
   - 关键词 keywords
   - 用户可能提出的问题 possible_questions
2. 使用 PostgreSQL `pg_trgm` 对 `content`、`source_section`、`search_text` 做文本相似度召回。
3. 用户提问时，先调用 DeepSeek 做查询改写和关键词抽取。
4. 数据库召回 TopN 候选片段。
5. 再调用 DeepSeek 对候选片段重排，选出 TopK。
6. 最后调用 DeepSeek 基于 TopK 片段生成答案并展示引用来源。

后续如果接入内网 embedding 模型，可以把检索层替换为 pgvector，RAG 回答层不需要大改。

---

## 3. 系统边界

系统需要实现：

1. 文档上传
2. 文档解析
3. 文档切片
4. 检索增强信息生成
5. 文档入库
6. 知识库检索
7. LLM 问答
8. 引用来源展示
9. 文档质量检查
10. 文档状态管理
11. Elasticsearch 日志源配置
12. 服务器日志源配置
13. 日志采样、检索和分析
14. 基于知识库文档的日志分析解释

系统暂不实现：

1. 自动执行命令
2. 自动修改生产配置
3. 自动重启服务
4. 自动清理数据
5. 多租户复杂权限
6. 工单系统深度集成
7. 实时日志流式采集和长期日志归档
8. 自动登录生产服务器执行修复命令

---

## 4. 核心业务流程

### 4.1 文档入库流程

```text
用户上传文档
    ↓
保存原始文件
    ↓
解析文档文本
    ↓
AI 检查文档质量
    ↓
切分为多个 chunk
    ↓
调用 DeepSeek 生成 chunk 摘要、关键词、可能问题
    ↓
写入 PostgreSQL，并使用 pg_trgm 建立文本检索索引
    ↓
文档状态变为：待审核
    ↓
人工审核通过
    ↓
文档状态变为：已发布
```

只有 `已发布` 状态的文档可以参与问答检索。

---

### 4.2 用户问答流程

```text
用户输入问题
    ↓
DeepSeek 查询改写和关键词抽取
    ↓
PostgreSQL pg_trgm 召回 TopN 文档片段
    ↓
按文档状态、系统、组件、文档类型过滤
    ↓
DeepSeek 对候选片段重排，选出 TopK
    ↓
组装 Prompt
    ↓
调用 DeepSeek v4
    ↓
返回答案
    ↓
展示引用来源
```

---

### 4.3 日志分析流程

```text
用户选择日志来源
    ↓
选择 Elasticsearch 查询或服务器日志路径
    ↓
填写时间范围、关键词、系统、组件、日志级别等条件
    ↓
后端读取日志样本
    ↓
对日志进行脱敏、截断、聚合和异常片段提取
    ↓
基于系统、组件、错误关键词检索已发布知识库文档
    ↓
组装“日志上下文 + 知识库片段 + 用户问题” Prompt
    ↓
调用 DeepSeek v4
    ↓
返回异常摘要、可能原因、排查建议、风险提示、引用文档和日志证据
```

日志分析只允许读取配置范围内的日志数据，不允许通过 LLM 自动生成并执行服务器命令。

---

## 5. 文档状态设计

文档状态包括：

```text
draft       草稿
reviewing   待审核
published   已发布
archived    已归档
deprecated  已废弃
rejected    已驳回
```

只有以下状态可以被知识库问答引用：

```text
published
```

---

## 6. 数据库设计

### 6.1 PostgreSQL 扩展

需要启用 `pg_trgm`，用于中文运维文档的近似文本检索：

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

---

### 6.2 知识库文档表

```sql
CREATE TABLE kb_document (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    file_type VARCHAR(50) NOT NULL,

    system_name VARCHAR(100),
    component_name VARCHAR(100),
    doc_type VARCHAR(100),

    version VARCHAR(50) DEFAULT 'v1.0',
    status VARCHAR(50) DEFAULT 'draft',

    tags TEXT,
    summary TEXT,

    quality_score INT DEFAULT 0,
    quality_result JSONB,

    created_by VARCHAR(100),
    reviewed_by VARCHAR(100),

    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    reviewed_at TIMESTAMP
);
```

字段说明：

```text
system_name    所属系统，例如：支付系统、核心系统、柜面系统
component_name 所属组件，例如：Redis、Nginx、K8s、TiDB
doc_type       文档类型，例如：启停手册、告警处置、应急预案、变更回滚
status         文档状态
quality_score  AI 质检分数
quality_result AI 质检结果 JSON
```

---

### 6.3 文档片段表

```sql
CREATE TABLE kb_chunk (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT NOT NULL REFERENCES kb_document(id) ON DELETE CASCADE,

    chunk_index INT NOT NULL,
    content TEXT NOT NULL,

    source_title VARCHAR(255),
    source_section VARCHAR(255),
    source_page INT,

    token_count INT DEFAULT 0,

    search_text TEXT,
    keywords JSONB,
    possible_questions JSONB,

    created_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
search_text          检索增强文本，由标题、章节、摘要、关键词、可能问题、正文组合而成
keywords             DeepSeek 从 chunk 中抽取的关键词 JSON
possible_questions   DeepSeek 生成的用户可能问题 JSON
```

---

### 6.4 问答记录表

```sql
CREATE TABLE qa_record (
    id BIGSERIAL PRIMARY KEY,

    question TEXT NOT NULL,
    answer TEXT NOT NULL,

    retrieved_chunks JSONB,
    model_name VARCHAR(100),

    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now()
);
```

---

### 6.5 文档审核记录表

```sql
CREATE TABLE kb_review_record (
    id BIGSERIAL PRIMARY KEY,

    document_id BIGINT NOT NULL REFERENCES kb_document(id) ON DELETE CASCADE,
    from_status VARCHAR(50),
    to_status VARCHAR(50),

    reviewer VARCHAR(100),
    comment TEXT,

    created_at TIMESTAMP DEFAULT now()
);
```

---

### 6.6 日志源配置表

日志源包括 Elasticsearch 和服务器文件两类。

```sql
CREATE TABLE log_source (
    id BIGSERIAL PRIMARY KEY,

    name VARCHAR(120) NOT NULL,
    source_type VARCHAR(50) NOT NULL,

    system_name VARCHAR(100),
    component_name VARCHAR(100),
    environment VARCHAR(50),

    endpoint TEXT,
    username VARCHAR(255),
    credential_ref TEXT,

    es_index_pattern VARCHAR(255),
    es_time_field VARCHAR(100),

    server_host VARCHAR(255),
    server_port INT DEFAULT 22,
    auth_type VARCHAR(50),
    log_path TEXT,
    path_allowlist JSONB,

    enabled BOOLEAN DEFAULT true,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
source_type       日志源类型：elasticsearch 或 server_file
endpoint          Elasticsearch 地址，例如：https://es.internal.local:9200
username          Elasticsearch 或服务器登录用户名
credential_ref    加密后的凭据引用，不保存明文密码或明文私钥
es_index_pattern  ES 索引模式，例如：app-log-*
es_time_field     ES 时间字段，例如：@timestamp
server_host       服务器地址
server_port       SSH 端口，默认 22
auth_type         password 或 private_key
log_path          默认日志文件路径，例如：/data/app/logs/error.log
path_allowlist    允许读取的目录或文件前缀 JSON
```

安全要求：

```text
1. password、private_key、private_key_passphrase 必须加密保存。
2. API 返回日志源配置时不得返回 credential_ref 的明文内容。
3. 服务器日志只能读取 path_allowlist 覆盖的路径。
4. 禁止读取 /etc、/root、/home/*/.ssh 等敏感目录。
5. 禁止通过日志源配置执行任意 shell 命令。
```

---

### 6.7 日志分析任务表

```sql
CREATE TABLE log_analysis_task (
    id BIGSERIAL PRIMARY KEY,

    source_id BIGINT REFERENCES log_source(id),
    question TEXT,

    system_name VARCHAR(100),
    component_name VARCHAR(100),
    time_start TIMESTAMP,
    time_end TIMESTAMP,
    keyword TEXT,
    log_level VARCHAR(50),

    status VARCHAR(50) DEFAULT 'pending',
    sample_count INT DEFAULT 0,
    error_message TEXT,

    result JSONB,
    retrieved_chunks JSONB,

    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
status            pending、running、success、failed
result            LLM 日志分析结果 JSON
retrieved_chunks  本次分析引用的知识库片段
sample_count      进入 LLM 分析的日志条数或片段数
```

---

## 7. 后端目录结构

```text
backend/
  cmd/
    server/
      main.go

  internal/
    config/
      config.go

    router/
      router.go

    middleware/
      cors.go
      logger.go
      recover.go

    model/
      kb_document.go
      kb_chunk.go
      qa_record.go
      review_record.go
      log_source.go
      log_analysis_task.go

    handler/
      document_handler.go
      qa_handler.go
      review_handler.go
      log_source_handler.go
      log_analysis_handler.go
      health_handler.go

    service/
      document_service.go
      parser_service.go
      chunk_service.go
      rag_service.go
      quality_service.go
      retrieval_metadata_service.go
      review_service.go
      log_source_service.go
      log_analysis_service.go

    repository/
      document_repository.go
      chunk_repository.go
      qa_repository.go
      log_source_repository.go
      log_analysis_repository.go

    client/
      deepseek_client.go
      elasticsearch_client.go
      ssh_log_client.go

    security/
      credential_crypto.go

    dto/
      document_dto.go
      qa_dto.go
      review_dto.go
      log_dto.go

    util/
      file_util.go
      text_util.go
      id_util.go

  migrations/
    001_init.sql

  go.mod
  go.sum
```

---

## 8. 前端目录结构

```text
frontend/
  src/
    main.tsx
    App.tsx

    api/
      http.ts
      documentApi.ts
      qaApi.ts
      reviewApi.ts
      logApi.ts

    pages/
      DashboardPage.tsx
      DocumentListPage.tsx
      DocumentUploadPage.tsx
      DocumentDetailPage.tsx
      ChatPage.tsx
      ReviewPage.tsx
      LogSourcePage.tsx
      LogAnalysisPage.tsx

    components/
      layout/
        AppLayout.tsx
        Sidebar.tsx
        Header.tsx

      document/
        DocumentTable.tsx
        DocumentUploadForm.tsx
        DocumentStatusBadge.tsx
        DocumentQualityCard.tsx

      chat/
        ChatWindow.tsx
        ChatInput.tsx
        AnswerCard.tsx
        CitationList.tsx

      review/
        ReviewPanel.tsx

      log/
        LogSourceForm.tsx
        LogSourceTable.tsx
        LogAnalysisForm.tsx
        LogAnalysisResult.tsx

    lib/
      utils.ts

  package.json
  vite.config.ts
  tailwind.config.js
```

---

## 9. 环境变量

### 9.1 后端环境变量

```env
APP_ENV=dev
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ops_kb
DB_SSLMODE=disable

FILE_STORAGE_TYPE=local
LOCAL_FILE_DIR=./data/uploads

DEEPSEEK_BASE_URL=http://deepseek-v4.internal.local/v1
DEEPSEEK_API_KEY=local-key
DEEPSEEK_MODEL=deepseek-v4

RAG_TOP_K=5
RAG_RECALL_K=30

CREDENTIAL_ENCRYPTION_KEY=change-me-32-bytes-minimum
LOG_SAMPLE_MAX_LINES=500
LOG_SAMPLE_MAX_BYTES=262144
LOG_ANALYSIS_TIMEOUT_SECONDS=60
SSH_CONNECT_TIMEOUT_SECONDS=10
ES_QUERY_TIMEOUT_SECONDS=15
```

说明：

如果内网 DeepSeek v4 接口兼容 OpenAI Chat Completions，则后端直接使用：

```text
POST /v1/chat/completions
```

如果不兼容，需要在 `deepseek_client.go` 中封装适配器。

日志源账号、密码、私钥不允许写死在配置文件或代码中。用户在页面录入后，后端使用 `CREDENTIAL_ENCRYPTION_KEY` 加密存储，并只在连接 Elasticsearch 或 SSH/SFTP 时临时解密使用。

---

## 10. API 设计

### 10.1 健康检查

```http
GET /api/health
```

返回：

```json
{
  "status": "ok"
}
```

---

### 10.2 上传文档

```http
POST /api/documents/upload
Content-Type: multipart/form-data
```

参数：

```text
file            文档文件
title           文档标题
systemName      所属系统
componentName   所属组件
docType         文档类型
tags            标签
```

返回：

```json
{
  "id": 1,
  "title": "Redis 内存告警处置手册",
  "status": "reviewing",
  "qualityScore": 86
}
```

---

### 10.3 查询文档列表

```http
GET /api/documents?page=1&pageSize=10&status=published
```

返回：

```json
{
  "items": [
    {
      "id": 1,
      "title": "Redis 内存告警处置手册",
      "systemName": "支付系统",
      "componentName": "Redis",
      "docType": "告警处置",
      "status": "published",
      "qualityScore": 92,
      "updatedAt": "2026-06-30T12:00:00Z"
    }
  ],
  "total": 1
}
```

---

### 10.4 查询文档详情

```http
GET /api/documents/{id}
```

返回：

```json
{
  "id": 1,
  "title": "Redis 内存告警处置手册",
  "summary": "本文档描述 Redis 内存告警的排查和处置流程。",
  "status": "published",
  "qualityScore": 92,
  "qualityResult": {
    "problems": [],
    "suggestions": []
  }
}
```

---

### 10.5 审核文档

```http
POST /api/documents/{id}/review
Content-Type: application/json
```

请求：

```json
{
  "action": "approve",
  "comment": "内容完整，可以发布"
}
```

action 支持：

```text
approve
reject
archive
deprecate
```

---

### 10.6 知识库问答

```http
POST /api/qa/ask
Content-Type: application/json
```

请求：

```json
{
  "question": "Redis 内存告警怎么处理？",
  "systemName": "支付系统",
  "componentName": "Redis",
  "topK": 5
}
```

返回：

```json
{
  "answer": "根据知识库中的 Redis 内存告警处置手册，建议先查看 info memory...",
  "citations": [
    {
      "documentId": 1,
      "documentTitle": "Redis 内存告警处置手册",
      "chunkId": 12,
      "sourceSection": "3. 排查步骤",
      "content": "Redis 内存使用率超过 85% 时，首先执行 info memory..."
    }
  ]
}
```

---

### 10.7 日志源管理

#### 10.7.1 创建日志源

```http
POST /api/log-sources
Content-Type: application/json
```

Elasticsearch 请求示例：

```json
{
  "name": "支付系统生产 ES",
  "sourceType": "elasticsearch",
  "systemName": "支付系统",
  "componentName": "payment-api",
  "environment": "prod",
  "endpoint": "https://es.internal.local:9200",
  "username": "elastic_user",
  "password": "******",
  "esIndexPattern": "payment-api-*",
  "esTimeField": "@timestamp"
}
```

服务器文件请求示例：

```json
{
  "name": "支付系统应用日志",
  "sourceType": "server_file",
  "systemName": "支付系统",
  "componentName": "payment-api",
  "environment": "prod",
  "serverHost": "10.10.1.20",
  "serverPort": 22,
  "username": "ops_reader",
  "authType": "private_key",
  "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----...",
  "privateKeyPassphrase": "******",
  "logPath": "/data/payment-api/logs/app.log",
  "pathAllowlist": ["/data/payment-api/logs/"]
}
```

说明：

```text
sourceType 支持 elasticsearch、server_file
authType 支持 password、private_key
password、privateKey、privateKeyPassphrase 只在创建或更新时提交
接口返回时不返回明文密码或私钥
```

#### 10.7.2 查询日志源列表

```http
GET /api/log-sources
```

#### 10.7.3 更新日志源

```http
PUT /api/log-sources/{id}
Content-Type: application/json
```

#### 10.7.4 删除日志源

```http
DELETE /api/log-sources/{id}
```

#### 10.7.5 测试日志源连接

```http
POST /api/log-sources/{id}/test
```

返回：

```json
{
  "ok": true,
  "message": "连接成功"
}
```

---

### 10.8 日志预览

```http
POST /api/logs/preview
Content-Type: application/json
```

请求：

```json
{
  "sourceId": 1,
  "timeStart": "2026-07-05T09:00:00+08:00",
  "timeEnd": "2026-07-05T10:00:00+08:00",
  "keyword": "ERROR timeout",
  "logLevel": "ERROR",
  "logPath": "/data/payment-api/logs/app.log",
  "limit": 100
}
```

返回：

```json
{
  "items": [
    {
      "timestamp": "2026-07-05T09:12:33+08:00",
      "level": "ERROR",
      "message": "request timeout, traceId=abc123",
      "source": "payment-api",
      "raw": "2026-07-05 09:12:33 ERROR request timeout, traceId=abc123"
    }
  ],
  "total": 42
}
```

---

### 10.9 日志分析

```http
POST /api/log-analysis
Content-Type: application/json
```

请求：

```json
{
  "sourceId": 1,
  "question": "支付接口 9 点后超时增多，可能是什么原因？",
  "systemName": "支付系统",
  "componentName": "payment-api",
  "timeStart": "2026-07-05T09:00:00+08:00",
  "timeEnd": "2026-07-05T10:00:00+08:00",
  "keyword": "timeout OR ERROR",
  "logLevel": "ERROR",
  "logPath": "/data/payment-api/logs/app.log",
  "topK": 5
}
```

返回：

```json
{
  "taskId": 1001,
  "status": "success",
  "summary": "09:12 后 payment-api 超时日志明显增加，主要集中在调用 Redis 获取风控配置阶段。",
  "possibleCauses": [
    "Redis 响应变慢或连接池耗尽",
    "上游请求量突增导致线程池排队"
  ],
  "evidence": [
    "09:12:33 ERROR request timeout, redis get risk_config timeout"
  ],
  "suggestions": [
    "检查 Redis 慢查询和连接数",
    "查看 payment-api 线程池和连接池指标",
    "按变更流程确认 09:00 前后是否有发布或配置变更"
  ],
  "riskTips": [
    "不要直接重启生产服务；如需重启必须走变更审批。"
  ],
  "citations": [
    {
      "documentId": 1,
      "documentTitle": "Redis 内存告警处置手册",
      "sourceSection": "3. 排查步骤"
    }
  ]
}
```

---

## 11. 文档切片规则

### 11.1 切片目标

每个 chunk 应该语义完整，便于检索。

### 11.2 默认参数

```text
chunk_size: 800 中文字符
chunk_overlap: 100 中文字符
```

### 11.3 切片优先级

优先按 Markdown 标题切分：

```text
# 一级标题
## 二级标题
### 三级标题
```

如果不是 Markdown，则按以下规则切分：

```text
1. 空行
2. 段落
3. 句号
4. 固定长度
```

每个 chunk 需要保存：

```text
document_id
chunk_index
content
source_title
source_section
source_page
```

---

## 12. RAG 检索规则

### 12.1 检索过滤条件

默认只检索：

```sql
kb_document.status = 'published'
```

如果用户指定系统或组件，需要增加过滤：

```sql
system_name = ?
component_name = ?
doc_type = ?
```

### 12.2 TopK

默认：

```text
topK = 5
```

### 12.3 召回和重排

第一阶段没有 embedding 模型，因此不使用向量相似度。

检索流程：

```text
1. DeepSeek 将用户问题改写为 query，并抽取 keywords
2. PostgreSQL 使用 pg_trgm 从 content、source_section、search_text 召回 TopN
3. DeepSeek 对 TopN 候选片段重排
4. 取 TopK 进入最终 RAG Prompt
```

示例 SQL：

```sql
SELECT
    c.id,
    c.document_id,
    c.content,
    c.source_section,
    d.title,
    d.system_name,
    d.component_name,
    d.doc_type,
    GREATEST(
        similarity(c.content, $1),
        similarity(COALESCE(c.search_text, ''), $1),
        similarity(COALESCE(c.source_section, ''), $1),
        similarity(d.title, $1)
    ) AS score
FROM kb_chunk c
JOIN kb_document d ON c.document_id = d.id
WHERE d.status = 'published'
  AND (
      c.content ILIKE '%' || $1 || '%'
      OR COALESCE(c.search_text, '') ILIKE '%' || $1 || '%'
      OR COALESCE(c.source_section, '') ILIKE '%' || $1 || '%'
      OR d.title ILIKE '%' || $1 || '%'
  )
ORDER BY score DESC
LIMIT $2;
```

---

## 13. 日志检索和分析规则

### 13.1 Elasticsearch 日志读取

Elasticsearch 日志源需要支持：

```text
1. 使用 endpoint、username、password 连接 ES。
2. 支持 HTTPS，允许配置是否跳过自签名证书校验，默认不跳过。
3. 根据 es_index_pattern 查询索引。
4. 根据 es_time_field、timeStart、timeEnd 做时间范围过滤。
5. 支持 keyword、logLevel、systemName、componentName 等过滤条件。
6. 查询结果按时间倒序或相关度排序，默认限制最大返回条数。
7. 对日志 message、stack_trace、raw 等字段做统一抽取。
```

ES 查询需要限制：

```text
1. 单次查询时间窗口默认不超过 24 小时。
2. 单次进入 LLM 的日志不超过 LOG_SAMPLE_MAX_LINES。
3. 单次进入 LLM 的日志总字节不超过 LOG_SAMPLE_MAX_BYTES。
4. 查询超时由 ES_QUERY_TIMEOUT_SECONDS 控制。
```

---

### 13.2 服务器日志文件读取

服务器日志源需要支持：

```text
1. 使用 SSH/SFTP 连接服务器。
2. 支持 username + password 认证。
3. 支持 username + private_key 认证。
4. 支持 private_key_passphrase。
5. 仅读取 log_path 或 path_allowlist 允许的文件。
6. 支持按关键词、日志级别、时间范围做本地过滤。
7. 支持 tail 最近 N 行，避免读取超大文件。
```

服务器日志读取限制：

```text
1. 禁止执行任意 shell 命令读取日志。
2. 优先使用 SFTP 读取文件内容；如必须 tail，只允许后端内置固定命令模板，不允许用户输入命令。
3. log_path 必须是绝对路径。
4. log_path 不能包含 .. 路径穿越。
5. 读取文件大小和行数必须受 LOG_SAMPLE_MAX_LINES、LOG_SAMPLE_MAX_BYTES 限制。
6. SSH 连接超时由 SSH_CONNECT_TIMEOUT_SECONDS 控制。
```

---

### 13.3 日志采样和脱敏

日志进入 LLM 前必须做预处理：

```text
1. 去除重复日志。
2. 聚合相同错误模板和出现次数。
3. 保留首条、末条、典型样本。
4. 脱敏手机号、身份证号、银行卡号、token、session、password、secret、access_key 等敏感字段。
5. 对超长堆栈做截断，保留异常类型、关键调用栈、错误消息。
6. 记录采样策略，便于用户判断分析结果可信度。
```

---

### 13.4 日志分析知识库召回

日志分析需要同时使用日志内容和知识库文档：

```text
1. 根据用户选择的 systemName、componentName、keyword 召回文档片段。
2. 从日志样本中抽取错误码、异常类名、接口名、组件名作为检索关键词。
3. 默认只召回 published 文档。
4. 将 TopK 文档片段和日志样本一起交给 LLM。
5. 返回结果必须区分“日志证据”和“知识库依据”。
```

---

## 14. LLM Prompt 设计

### 14.1 知识库问答 Prompt

```text
你是一个资深银行生产运维专家。

请严格基于【知识库内容】回答用户问题。

要求：
1. 不要编造知识库中不存在的信息。
2. 如果知识库没有相关依据，请明确说明：“知识库中未找到明确依据”。
3. 涉及生产命令时，只能作为排查建议展示，不允许描述为可以直接执行。
4. 涉及重启、删除、清理、扩容、切换、回滚等高风险操作时，必须提示需要按生产变更流程审批。
5. 回答要结构清晰。
6. 最后列出引用来源。

用户问题：
{{question}}

知识库内容：
{{chunks}}

请按以下格式回答：

## 结论

## 依据

## 排查步骤

## 建议命令

## 风险提示

## 引用来源
```

---

### 14.2 文档质量检查 Prompt

```text
你是一个银行生产运维文档审核专家。

请检查下面的运维手册是否适合进入生产知识库。

请从以下维度评分，总分 100 分：

1. 完整性，30 分
   - 是否包含适用系统、适用环境、适用场景
   - 是否包含操作步骤、验证步骤、回滚步骤、风险说明

2. 准确性，30 分
   - 命令、路径、端口、服务名是否清晰
   - 是否存在明显矛盾或过期描述
   - 是否区分生产、测试、灾备环境

3. 可操作性，20 分
   - 一线运维人员是否可以照着执行
   - 步骤是否有顺序
   - 是否避免“视情况处理”等模糊表达

4. 可验证性，10 分
   - 每个关键步骤是否有验证方法
   - 是否写明正常结果

5. 可追溯性，10 分
   - 是否包含版本号、更新时间、责任人、审核人

请输出 JSON，不要输出多余解释。

输出格式：

{
  "score": 0,
  "level": "pass | warning | reject",
  "summary": "",
  "problems": [
    {
      "type": "",
      "description": "",
      "suggestion": ""
    }
  ],
  "missingFields": [],
  "riskPoints": [],
  "rewriteSuggestions": []
}

手册内容：
{{document_content}}
```

---

### 14.3 日志分析 Prompt

```text
你是一个资深银行生产运维日志分析专家。

请基于【日志样本】和【知识库内容】分析用户问题。

要求：
1. 必须区分日志中可以直接观察到的事实、基于知识库的依据、以及推测性的可能原因。
2. 不要编造日志中不存在的错误、时间点、接口、主机或指标。
3. 如果知识库没有相关依据，请明确说明：“知识库中未找到明确依据”。
4. 涉及生产命令时，只能作为排查建议展示，不允许描述为可以直接执行。
5. 涉及重启、删除、清理、扩容、切换、回滚等高风险操作时，必须提示需要按生产变更流程审批。
6. 输出需要包含日志证据和引用文档。

用户问题：
{{question}}

日志来源：
{{log_source}}

日志时间范围：
{{time_range}}

日志样本：
{{log_samples}}

知识库内容：
{{chunks}}

请按以下格式回答：

## 异常摘要

## 日志证据

## 可能原因

## 建议排查步骤

## 风险提示

## 知识库引用
```

---

## 15. DeepSeek v4 客户端要求

### 15.1 Chat 接口

实现文件：

```text
backend/internal/client/deepseek_client.go
```

需要提供方法：

```go
type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatRequest struct {
    Model       string        `json:"model"`
    Messages    []ChatMessage `json:"messages"`
    Temperature float64       `json:"temperature"`
}

type ChatResponse struct {
    Content string
}

type DeepSeekClient interface {
    Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error)
}
```

如果接口 OpenAI 兼容，请调用：

```http
POST {DEEPSEEK_BASE_URL}/chat/completions
```

---

### 15.2 LLM 用途

由于当前只有 DeepSeek v4 LLM，后端需要复用同一个 Chat 接口完成：

```text
1. 文档质量检查
2. chunk 检索增强信息生成
3. 用户问题查询改写和关键词抽取
4. 候选 chunk 重排
5. 最终 RAG 回答生成
6. 日志异常摘要和根因分析
7. 日志关键词、错误码、异常类型抽取
```

---

## 16. 外部日志客户端要求

### 16.1 ElasticsearchClient

实现文件：

```text
backend/internal/client/elasticsearch_client.go
```

需要提供方法：

```go
type ESLogQuery struct {
    Endpoint     string
    Username     string
    Password     string
    IndexPattern string
    TimeField    string
    TimeStart    time.Time
    TimeEnd      time.Time
    Keyword      string
    LogLevel     string
    Limit        int
}

type LogItem struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Source    string    `json:"source"`
    Raw       string    `json:"raw"`
}

type ElasticsearchClient interface {
    Test(ctx context.Context, cfg ESConfig) error
    QueryLogs(ctx context.Context, query ESLogQuery) ([]LogItem, error)
}
```

---

### 16.2 SSHLogClient

实现文件：

```text
backend/internal/client/ssh_log_client.go
```

需要提供方法：

```go
type SSHLogQuery struct {
    Host          string
    Port          int
    Username      string
    Password      string
    PrivateKey    string
    Passphrase    string
    AuthType      string
    LogPath       string
    PathAllowlist []string
    TimeStart     time.Time
    TimeEnd       time.Time
    Keyword       string
    LogLevel      string
    Limit         int
}

type SSHLogClient interface {
    Test(ctx context.Context, cfg SSHConfig) error
    ReadLogs(ctx context.Context, query SSHLogQuery) ([]LogItem, error)
}
```

---

## 17. 后端 Service 职责

### 17.1 DocumentService

负责：

```text
1. 保存上传文件
2. 创建文档记录
3. 调用 ParserService 解析文本
4. 调用 QualityService 质检
5. 调用 ChunkService 切片
6. 调用 RetrievalMetadataService 生成检索增强信息
7. 保存 chunk
8. 更新文档状态
```

---

### 17.2 ParserService

负责解析：

```text
.txt
.md
.pdf
.doc
.docx
.xls
.xlsx
```

第一版实现：

```text
.txt
.md
.doc
.docx
.xls
.xlsx
```

Office 文件优先通过 Go 库解析文本，不直接手写压缩包 XML 解析逻辑；`.docx/.xlsx/.xls` 支持解析，`.doc` 老二进制 Word 文件会提示先转换为 `.docx`。PDF 可以预留接口，后续扩展。

---

### 17.3 ChunkService

负责：

```text
1. 按标题切片
2. 按段落切片
3. 控制 chunk_size
4. 控制 chunk_overlap
5. 生成 chunk_index
```

---

### 17.4 RAGService

负责：

```text
1. 接收用户问题
2. 调用 DeepSeek 做查询改写和关键词抽取
3. 使用 pg_trgm 召回候选 chunk
4. 调用 DeepSeek 对候选 chunk 重排
5. 组装 Prompt
6. 调用 DeepSeek 生成答案
7. 保存问答记录
8. 返回答案和引用来源
```

---

### 17.5 RetrievalMetadataService

负责：

```text
1. 接收 chunk 内容
2. 调用 DeepSeek 生成摘要、关键词、可能问题
3. 生成 search_text
4. 写入 kb_chunk 的 search_text、keywords、possible_questions 字段
```

### 17.6 QualityService

负责：

```text
1. 调用 LLM 检查文档质量
2. 解析 JSON 结果
3. 写入 quality_score
4. 写入 quality_result
5. 根据分数决定初始状态
```

规则：

```text
score >= 90: reviewing
70 <= score < 90: reviewing，但是标记 warning
score < 70: draft，不允许直接提交审核
```

---

### 17.7 LogSourceService

负责：

```text
1. 创建和更新日志源配置
2. 校验 Elasticsearch 和服务器日志源参数
3. 加密保存 password、private_key、private_key_passphrase
4. 返回日志源列表时隐藏敏感凭据
5. 调用 ElasticsearchClient 或 SSHLogClient 测试连通性
6. 校验服务器日志路径是否在 path_allowlist 内
```

---

### 17.8 LogAnalysisService

负责：

```text
1. 根据日志源类型读取 Elasticsearch 或服务器日志文件
2. 按时间、关键词、日志级别过滤日志
3. 对日志做采样、聚合、脱敏和截断
4. 从日志中抽取错误码、异常类型、接口名、组件名
5. 调用 RAG 检索相关知识库文档
6. 组装日志分析 Prompt
7. 调用 DeepSeek v4 生成分析结果
8. 保存 log_analysis_task 记录
9. 返回日志证据、可能原因、排查建议和知识库引用
```

---

## 18. 前端页面说明

### 18.1 DashboardPage

展示：

```text
1. 文档总数
2. 已发布文档数
3. 待审核文档数
4. 平均质量分
5. 最近问答记录
```

---

### 18.2 DocumentUploadPage

功能：

```text
1. 上传文档
2. 填写标题
3. 选择所属系统
4. 选择组件
5. 选择文档类型
6. 填写标签
7. 提交入库
8. 展示 AI 质量检查结果
```

---

### 18.3 DocumentListPage

功能：

```text
1. 文档列表
2. 按状态筛选
3. 按系统筛选
4. 按组件筛选
5. 按文档类型筛选
6. 查看详情
```

---

### 18.4 ChatPage

功能：

```text
1. 用户输入问题
2. 可选系统过滤条件
3. 可选组件过滤条件
4. 展示 AI 回答
5. 展示引用文档
6. 展示引用片段
```

回答区域需要明确提示：

```text
AI 回答仅供运维排查参考，生产操作请遵守变更审批流程。
```

---

### 18.5 ReviewPage

功能：

```text
1. 查看待审核文档
2. 查看 AI 质检结果
3. 查看问题和修改建议
4. 审核通过
5. 审核驳回
6. 废弃文档
```

---

### 18.6 LogSourcePage

功能：

```text
1. 查看日志源列表
2. 新增 Elasticsearch 日志源
3. 新增服务器日志文件源
4. 支持账号密码认证
5. 支持 SSH 私钥认证和私钥口令
6. 测试日志源连接
7. 启用或禁用日志源
```

---

### 18.7 LogAnalysisPage

功能：

```text
1. 选择日志源
2. 填写时间范围、关键词、日志级别、日志路径
3. 预览日志样本
4. 输入分析问题
5. 提交日志分析
6. 展示异常摘要、日志证据、可能原因、排查建议
7. 展示知识库引用来源
8. 提示 AI 分析仅供排查参考
```

---

## 19. shadcn/ui 组件建议

使用以下组件：

```text
Button
Card
Input
Textarea
Select
Badge
Table
Tabs
Dialog
Alert
ScrollArea
Separator
Progress
```

页面布局：

```text
左侧 Sidebar
顶部 Header
中间 Content
```

---

## 20. 前端 API 封装

### 20.1 documentApi.ts

需要实现：

```ts
export async function uploadDocument(formData: FormData)

export async function listDocuments(params: {
  page: number
  pageSize: number
  status?: string
  systemName?: string
  componentName?: string
  docType?: string
})

export async function getDocument(id: number)

export async function reviewDocument(id: number, data: {
  action: 'approve' | 'reject' | 'archive' | 'deprecate'
  comment?: string
})
```

---

### 20.2 qaApi.ts

需要实现：

```ts
export async function askQuestion(data: {
  question: string
  systemName?: string
  componentName?: string
  docType?: string
  topK?: number
})
```

---

### 20.3 logApi.ts

需要实现：

```ts
export async function listLogSources()

export async function createLogSource(data: {
  name: string
  sourceType: 'elasticsearch' | 'server_file'
  systemName?: string
  componentName?: string
  environment?: string
  endpoint?: string
  username?: string
  password?: string
  esIndexPattern?: string
  esTimeField?: string
  serverHost?: string
  serverPort?: number
  authType?: 'password' | 'private_key'
  privateKey?: string
  privateKeyPassphrase?: string
  logPath?: string
  pathAllowlist?: string[]
})

export async function updateLogSource(id: number, data: Partial<LogSourceInput>)

export async function deleteLogSource(id: number)

export async function testLogSource(id: number)

export async function previewLogs(data: {
  sourceId: number
  timeStart?: string
  timeEnd?: string
  keyword?: string
  logLevel?: string
  logPath?: string
  limit?: number
})

export async function analyzeLogs(data: {
  sourceId: number
  question: string
  systemName?: string
  componentName?: string
  timeStart?: string
  timeEnd?: string
  keyword?: string
  logLevel?: string
  logPath?: string
  topK?: number
})
```

---

## 21. 后端返回格式统一

所有接口统一返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

错误返回：

```json
{
  "code": 500,
  "message": "错误信息",
  "data": null
}
```

---

## 22. 安全要求

1. 不允许系统自动执行 LLM 生成的命令。
2. 所有命令只能作为文本建议展示。
3. 涉及生产变更、重启、删除、清理、扩容、回滚时，必须提示审批。
4. 不允许把草稿、废弃、归档文档用于问答。
5. LLM 回答必须带引用来源。
6. 如果检索不到文档，必须明确说明知识库没有依据。
7. 文档上传大小需要限制，默认最大 50MB。
8. 文件类型需要白名单控制。
9. 后端需要记录问答日志，方便审计。
10. 日志源凭据必须加密保存，不允许返回明文密码或私钥。
11. 服务器日志读取必须限制在 path_allowlist 内。
12. 日志分析必须对敏感字段脱敏后再发送给 LLM。
13. 日志分析结果只能作为排查建议，不允许自动执行修复动作。
14. Elasticsearch 查询和服务器日志读取必须限制时间范围、返回行数和总字节数。

---

## 23. 第一阶段 MVP 范围

第一阶段只实现最小可用版本。

### 必须实现

```text
1. React + shadcn/ui 前端基础页面
2. Golang + Gin 后端服务
3. PostgreSQL + pg_trgm 表结构
4. Markdown / TXT 文档上传
5. 文档切片
6. 检索增强信息入库
7. 知识库问答
8. DeepSeek v4 调用
9. 引用来源展示
10. 文档质量检查
11. Elasticsearch 日志源配置
12. 服务器日志文件源配置
13. 日志预览
14. 基于日志和知识库的 LLM 分析
```

### 可以暂缓

```text
1. PDF 解析
2. MinIO
3. 用户登录
4. 权限管理
5. 文档版本对比
6. Wiki 自动同步
7. 工单系统集成
8. 日志长期归档
9. 实时流式日志监控
10. 自动化处置闭环
```

---

## 24. Codex 开发任务拆分

请按以下顺序实现。

### Task 1：初始化后端项目

目标：

```text
创建 Golang Gin 项目，完成基础启动、配置加载、健康检查接口。
```

验收标准：

```text
1. go run cmd/server/main.go 可以启动
2. GET /api/health 返回 {"status":"ok"}
3. 支持从 .env 读取配置
```

---

### Task 2：初始化前端项目

目标：

```text
创建 React + Vite + TypeScript + shadcn/ui 项目。
```

验收标准：

```text
1. pnpm dev 可以启动
2. 页面包含基础布局
3. 有 Sidebar 和 Header
```

---

### Task 3：数据库模型和迁移

目标：

```text
创建 kb_document、kb_chunk、qa_record、kb_review_record 表。
```

验收标准：

```text
1. migrations/001_init.sql 可执行
2. GORM model 与表结构对应
3. 后端启动时可以连接数据库
```

---

### Task 4：文档上传

目标：

```text
实现 /api/documents/upload。
```

验收标准：

```text
1. 支持上传 .md、.txt、.doc、.docx、.xls、.xlsx
2. 文件保存到本地目录
3. 创建 kb_document 记录
4. 返回文档 ID
```

---

### Task 5：文档解析和切片

目标：

```text
解析 .md、.txt、.doc、.docx、.xls、.xlsx 文档，切分为 chunks。
```

验收标准：

```text
1. 可以读取文档内容
2. 可以生成多个 chunk
3. chunk 内容不为空
4. chunk_index 顺序正确
```

---

### Task 6：检索增强信息入库

目标：

```text
调用 DeepSeek 为每个 chunk 生成摘要、关键词、可能问题，并写入 kb_chunk。
```

验收标准：

```text
1. 每个 chunk 都有 search_text
2. keywords、possible_questions 可以正常保存为 JSONB
3. pg_trgm 索引可以正常执行文本召回
```

---

### Task 7：DeepSeek v4 调用

目标：

```text
实现 deepseek_client.go。
```

验收标准：

```text
1. 可以发送 messages
2. 可以解析回答内容
3. 错误时返回明确 error
```

---

### Task 8：知识库问答

目标：

```text
实现 /api/qa/ask。
```

验收标准：

```text
1. 用户问题可以通过 DeepSeek 改写和抽取关键词
2. 可以用 pg_trgm 召回候选 chunks
3. 可以调用 DeepSeek 重排候选 chunks
4. 可以组装 Prompt
5. 可以调用 DeepSeek v4
6. 返回 answer 和 citations
```

---

### Task 9：前端文档上传页面

目标：

```text
实现 DocumentUploadPage。
```

验收标准：

```text
1. 可以选择文件
2. 可以填写文档元数据
3. 可以提交上传
4. 可以展示上传结果
```

---

### Task 10：前端问答页面

目标：

```text
实现 ChatPage。
```

验收标准：

```text
1. 可以输入问题
2. 可以提交问题
3. 可以展示 AI 回答
4. 可以展示引用来源
5. 页面提示 AI 仅供参考
```

---

### Task 11：文档质量检查

目标：

```text
上传文档后调用 LLM 检查文档质量。
```

验收标准：

```text
1. 返回 quality_score
2. 返回 quality_result
3. 前端可以展示问题和修改建议
```

---

### Task 12：文档审核

目标：

```text
实现文档审核发布流程。
```

验收标准：

```text
1. 待审核文档可以 approve
2. 通过后状态变为 published
3. 只有 published 文档参与问答
```

---

### Task 13：日志源配置

目标：

```text
实现 Elasticsearch 和服务器日志文件源配置。
```

验收标准：

```text
1. 可以创建 Elasticsearch 日志源，支持 endpoint、账号、密码、索引模式、时间字段
2. 可以创建服务器日志源，支持 host、port、账号密码、私钥、日志路径、路径白名单
3. 凭据加密保存，接口不返回明文密码或私钥
4. 可以测试日志源连接
```

---

### Task 14：日志读取和预览

目标：

```text
实现 /api/logs/preview。
```

验收标准：

```text
1. 可以从 Elasticsearch 按时间范围、关键词、日志级别读取日志
2. 可以从服务器指定路径读取日志文件
3. 服务器日志路径必须受 path_allowlist 限制
4. 返回日志样本前完成行数和字节数限制
5. 日志内容做基础脱敏
```

---

### Task 15：日志分析

目标：

```text
实现 /api/log-analysis，结合日志样本、知识库文档和 DeepSeek v4 生成分析结果。
```

验收标准：

```text
1. 可以从日志中抽取错误关键词、异常类型、接口名
2. 可以召回相关 published 文档片段
3. 可以组装日志分析 Prompt
4. 返回异常摘要、日志证据、可能原因、排查建议、风险提示、知识库引用
5. 保存 log_analysis_task 记录
```

---

### Task 16：前端日志页面

目标：

```text
实现 LogSourcePage 和 LogAnalysisPage。
```

验收标准：

```text
1. 可以管理 Elasticsearch 和服务器日志源
2. 可以测试连接
3. 可以选择日志源并预览日志
4. 可以提交日志分析
5. 可以展示日志证据、分析结论和知识库引用
6. 页面提示 AI 日志分析仅供排查参考
```

---

## 25. 开发约束

1. 代码需要清晰分层，不要把业务逻辑写在 handler 中。
2. handler 只负责参数解析和响应。
3. service 负责业务逻辑。
4. repository 负责数据库操作。
5. client 负责调用外部模型服务。
6. 所有错误需要返回明确错误信息。
7. 不要在代码里硬编码模型地址、数据库密码。
8. 所有配置从环境变量读取。
9. 前端 API 调用统一放在 src/api 目录。
10. 前端页面组件不要过度复杂，优先实现可用性。
11. 凭据加密、脱敏、路径白名单逻辑必须放在后端，不依赖前端校验。
12. 日志读取、日志采样、日志分析需要有明确超时和大小限制。

---

## 26. 最终验收标准

完成后，系统应该可以做到：

```text
1. 启动前端和后端
2. 上传一篇 Markdown 运维手册
3. 系统自动解析、切片、生成检索增强信息并入库
4. 审核发布文档
5. 用户在问答页面提问
6. 系统检索知识库
7. 调用内网 DeepSeek v4 生成答案
8. 页面展示答案和引用来源
9. 如果知识库无相关内容，明确提示没有依据
10. 配置 Elasticsearch 日志源并完成连通性测试
11. 配置服务器指定路径日志源，支持账号密码或私钥认证
12. 预览指定时间范围内的日志样本
13. 基于日志样本和已发布文档生成 LLM 日志分析结果
14. 日志分析结果展示日志证据、知识库引用和风险提示
```

---

## 27. 示例运维手册

可以使用以下内容测试。

````markdown
# Redis 内存告警处置手册

## 适用范围

适用于生产环境 Redis 实例内存使用率超过 85% 的告警场景。

## 告警含义

Redis 内存使用率过高，可能导致请求延迟升高、key 被淘汰，严重时可能影响业务写入。

## 排查步骤

### 1. 查看内存使用情况

执行命令：

```bash
redis-cli -h <host> -p <port> info memory
````

重点关注：

```text
used_memory
used_memory_human
used_memory_rss
maxmemory
mem_fragmentation_ratio
```

### 2. 检查 bigkey

执行命令：

```bash
redis-cli -h <host> -p <port> --bigkeys
```

如果发现大 key，需要联系业务确认是否可以清理或优化。

### 3. 查看慢查询

执行命令：

```bash
redis-cli -h <host> -p <port> slowlog get 10
```

## 处理建议

如果只是短时间内存升高，可以持续观察。

如果内存持续超过 90%，需要联系 DBA 或 Redis 负责人评估扩容、清理或调整淘汰策略。

## 风险提示

生产环境不允许直接删除 key。

涉及清理、扩容、重启、配置变更时，必须按照生产变更流程审批。

## 验证方法

处理后再次执行：

```bash
redis-cli -h <host> -p <port> info memory
```

确认内存使用率下降，业务访问正常，告警恢复。

## 回滚方案

如果调整配置后出现异常，应恢复原配置，并按生产变更流程执行回滚。

## 责任团队

基础运维团队、DBA 团队。

```
```

---

## 28. 示例问题

可以用以下问题测试：

```text
Redis 内存告警怎么处理？
```

期望回答应该包含：

```text
1. 告警含义
2. info memory 检查
3. bigkeys 检查
4. slowlog 检查
5. 生产环境不允许直接删除 key
6. 涉及清理、扩容、重启必须走变更审批
7. 引用 Redis 内存告警处置手册
```
