# 运维知识库 RAG 系统开发说明

## 1. 项目目标

开发一个面向运维场景的知识库问答系统。

系统基于内部运维文档、操作手册、告警处置手册、应急预案、变更回滚方案等资料，构建 RAG 知识库。

用户可以上传文档，并基于知识库提问。系统需要先检索知识库内容，再调用已配置的大模型接口生成答案，默认兼容内网 DeepSeek v4，也支持 Qwen3 和其他 OpenAI Chat Completions 兼容接口。

系统还需要支持对接 Elasticsearch 和服务器内指定路径的日志文件，结合已发布运维文档和可选 LLM 接口做日志检索、异常摘要、根因分析和处置建议。

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
* 可配置 LLM 接口：DeepSeek v4、Qwen3、OpenAI-compatible

### 检索策略

当前环境不依赖独立 embedding 模型。

因此第一阶段不使用 pgvector / embedding，改用以下方案尽量接近 RAG 效果：

1. 文档切片后，调用默认 LLM 为每个 chunk 生成检索增强信息：
   - 摘要 summary
   - 关键词 keywords
   - 用户可能提出的问题 possible_questions
2. 使用 PostgreSQL `pg_trgm` 对 `content`、`source_section`、`search_text` 做文本相似度召回。
3. 用户提问时，先调用默认 LLM 做查询改写和关键词抽取。
4. 数据库召回 TopN 候选片段。
5. 再调用默认 LLM 对候选片段重排，选出 TopK。
6. 最后调用默认 LLM 基于 TopK 片段生成答案并展示引用来源。

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
调用默认 LLM
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
调用默认 LLM
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

`DEEPSEEK_*` 作为无数据库配置时的兜底默认模型接口。正式使用时，可以在“模型接口”页面配置 DeepSeek、Qwen3 或其他 OpenAI Chat Completions 兼容接口，并选择一个启用的默认模型。

如果模型接口兼容 OpenAI Chat Completions，则后端直接使用：

```text
POST /v1/chat/completions
```

如果不兼容，需要在 LLM client 中封装适配器。

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

### 10.10 模型接口管理

#### 10.10.1 查询模型接口列表

```http
GET /api/llm-configs
```

#### 10.10.2 创建模型接口

```http
POST /api/llm-configs
Content-Type: application/json
```

请求：

```json
{
  "name": "Qwen3 生产接口",
  "provider": "qwen3",
  "baseUrl": "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "model": "qwen3-plus",
  "apiKey": "******",
  "temperature": 0.2,
  "isDefault": true,
  "enabled": true
}
```

说明：

```text
provider 支持 deepseek、qwen3、openai_compatible
apiKey 只在创建或更新时提交，后端加密保存，查询接口不返回明文
同一时间只有一个默认模型接口
所有文档质检、检索增强、问答、日志分析默认使用当前默认模型接口
未配置默认模型时，回退使用 DEEPSEEK_* 环境变量
```

#### 10.10.3 更新模型接口

```http
PUT /api/llm-configs/{id}
```

#### 10.10.4 设为默认模型接口

```http
POST /api/llm-configs/{id}/default
```

#### 10.10.5 测试模型接口

```http
POST /api/llm-configs/{id}/test
Content-Type: application/json
```

请求：

```json
{
  "prompt": "请回复：连接成功"
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

## 15. LLM 客户端要求

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
    Model string
}

type DeepSeekClient interface {
    Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error)
}
```

DeepSeek、Qwen3、OpenAI-compatible 均优先按 OpenAI Chat Completions 兼容接口调用：

```http
POST {BASE_URL}/chat/completions
```

Qwen3 示例配置：

```text
provider = qwen3
baseUrl  = https://dashscope.aliyuncs.com/compatible-mode/v1
model    = qwen3-plus 或内网实际模型名
apiKey   = 页面录入后加密保存
```

---

### 15.2 LLM 用途

后端需要复用当前默认 LLM 接口完成：

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
7. 调用默认 LLM 生成分析结果
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
8. 可选 LLM 接口配置和调用
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

### Task 7：可选 LLM 接口调用

目标：

```text
实现 OpenAI Chat Completions 兼容 LLM 客户端，支持 DeepSeek、Qwen3 和 OpenAI-compatible 配置。
```

验收标准：

```text
1. 可以发送 messages
2. 可以解析回答内容
3. 错误时返回明确 error
4. 可以通过页面新增 Qwen3 接口
5. 可以选择默认模型接口
6. 未配置默认接口时回退使用 DEEPSEEK_* 环境变量
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
5. 可以调用默认 LLM
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
7. 调用默认 LLM 生成答案
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

---

## 29. K8s 日志 / 告警采集与分析扩展说明

### 29.1 扩展目标

在现有运维知识库 RAG 系统和日志分析能力基础上，增加 Kubernetes 集群只读采集能力。

系统需要支持对接 Kubernetes API Server，读取指定集群、指定 namespace 下的 Pod、Event、Workload、Service、Endpoint、Ingress、HPA、PVC 等信息，并结合 Prometheus 指标、Elasticsearch 日志、服务器日志文件和已发布知识库文档，完成 K8s 告警解释、Pod 异常诊断、Ingress 访问异常分析、Node 异常分析和服务访问链路分析。

本功能只用于辅助运维分析，不自动执行任何生产命令，不自动修改 Kubernetes 资源，不自动删除、重启、扩容、缩容、回滚任何业务组件。

---

### 29.2 K8s 采集能力边界

系统需要实现：

1. Kubernetes 集群配置管理
2. Kubernetes 连接测试
3. Namespace 只读资源采集
4. Pod 状态采集
5. Pod Events 采集
6. Pod 当前日志采集
7. Pod previous 日志采集
8. Deployment / StatefulSet / DaemonSet / ReplicaSet 采集
9. Service / Endpoint / EndpointSlice 采集
10. Ingress 采集
11. HPA 采集
12. PVC 采集
13. 可选 Node 只读信息采集
14. Alertmanager 告警内容解析
15. 基于告警 labels 自动补充 K8s 上下文
16. 基于 K8s 上下文和知识库文档的 LLM 分析
17. K8s 诊断历史记录保存

系统暂不实现：

1. 自动执行 kubectl 命令
2. 自动 exec 进入容器
3. 自动 attach 容器
4. 自动 port-forward
5. 自动 delete Pod
6. 自动 rollout restart
7. 自动 scale
8. 自动 patch / apply / edit Kubernetes 资源
9. 自动读取 Secret 明文
10. 自动修改 ConfigMap
11. 自动修改 Ingress、Service、Deployment
12. 自动进行生产修复动作

---

### 29.3 K8s 日志 / 告警分析流程

#### 29.3.1 告警接入流程

```text
用户粘贴 Alertmanager 告警或 Webhook 接收告警
    ↓
解析 alertname、cluster、namespace、pod、container、deployment、service、ingress、node 等 labels
    ↓
根据告警类型判断需要采集的 K8s 上下文
    ↓
调用 Kubernetes API 读取只读资源
    ↓
可选调用 Prometheus 查询最近 30 分钟指标
    ↓
可选调用 Elasticsearch / 服务器日志源查询应用日志
    ↓
对日志和资源信息进行脱敏、截断和结构化
    ↓
根据 systemName、componentName、alertName、error keywords 检索已发布知识库文档
    ↓
组装“K8s 告警上下文 + 日志样本 + 指标摘要 + 知识库片段 + 用户问题” Prompt
    ↓
调用默认 LLM
    ↓
返回异常摘要、关键证据、可能原因、建议排查步骤、风险提示、知识库引用
```

---

#### 29.3.2 Pod 诊断流程

```text
用户输入 cluster、namespace、pod、container
    ↓
读取 Pod 基本状态
    ↓
读取 Pod containerStatuses
    ↓
读取 Pod Events
    ↓
读取当前容器日志
    ↓
读取 previous 容器日志
    ↓
反查所属 Deployment / StatefulSet / DaemonSet / Job
    ↓
读取 ReplicaSet 状态
    ↓
读取 Service / Endpoint 关联情况
    ↓
可选读取 Node 状态
    ↓
可选读取 Prometheus CPU、内存、重启次数指标
    ↓
检索已发布知识库文档
    ↓
调用 LLM 生成 Pod 诊断报告
```

---

#### 29.3.3 Ingress / Service 诊断流程

```text
用户输入 cluster、namespace、ingress 或 service
    ↓
读取 Ingress 规则
    ↓
读取后端 Service
    ↓
读取 Service selector 和 ports
    ↓
读取 Endpoint / EndpointSlice
    ↓
反查后端 Pod 列表
    ↓
检查 Pod Ready 状态
    ↓
可选读取 Nginx Ingress Controller 日志
    ↓
可选读取 Prometheus 5xx、499、请求耗时、QPS 指标
    ↓
检索已发布知识库文档
    ↓
调用 LLM 生成访问异常分析
```

---

### 29.4 K8s 需要采集的内容

#### 29.4.1 Pod 状态

需要采集字段：

```text
cluster
namespace
pod_name
phase
node_name
pod_ip
host_ip
start_time
labels
annotations keys
ownerReferences
restart_policy
service_account_name
```

containerStatuses 需要采集：

```text
container_name
image
image_id
ready
restart_count
state
state_reason
state_message
last_state
last_state_reason
last_state_message
last_state_exit_code
last_state_started_at
last_state_finished_at
```

重点识别以下状态：

```text
CrashLoopBackOff
ImagePullBackOff
ErrImagePull
CreateContainerConfigError
CreateContainerError
RunContainerError
OOMKilled
Error
Completed
Pending
Evicted
ContainerCreating
Terminating
```

---

#### 29.4.2 Pod Events

需要采集字段：

```text
type
reason
message
count
first_timestamp
last_timestamp
reporting_component
involved_object_kind
involved_object_name
```

重点关注：

```text
FailedScheduling
FailedMount
FailedAttachVolume
BackOff
Unhealthy
Killing
Pulled
Pulling
Failed
Created
Started
Evicted
Preempting
NodeNotReady
```

---

#### 29.4.3 Pod 日志

需要支持：

```text
当前日志 logs
上一次容器日志 previous logs
按 container 查询日志
限制 tail 行数
限制总字节数
脱敏后进入 LLM
```

默认限制：

```text
K8S_LOG_TAIL_LINES=300
K8S_LOG_MAX_BYTES=262144
K8S_LOG_PREVIOUS_ENABLED=true
```

日志进入 LLM 前需要：

```text
1. 去除重复日志
2. 聚合相同错误模板
3. 保留首条、末条、典型样本
4. 截断超长堆栈
5. 脱敏手机号、身份证号、银行卡号、token、password、secret、access_key、authorization、cookie 等字段
```

---

#### 29.4.4 Workload 信息

需要采集以下资源：

```text
Deployment
StatefulSet
DaemonSet
ReplicaSet
Job
CronJob
```

Deployment / StatefulSet / DaemonSet 需要采集：

```text
name
namespace
replicas
ready_replicas
available_replicas
updated_replicas
strategy
selector
template labels
containers
images
resources.requests
resources.limits
env keys
envFrom refs
volumeMounts
volumes refs
readinessProbe
livenessProbe
startupProbe
nodeSelector
affinity
tolerations
```

注意：

```text
1. env 只采集 key，不采集 value。
2. SecretRef 只采集 Secret 名称和 key 名称，不采集 Secret value。
3. ConfigMapRef 默认只采集名称，不读取内容。
4. 如需读取 ConfigMap 内容，必须单独配置权限，并在进入 LLM 前脱敏。
```

---

#### 29.4.5 Service / Endpoint / EndpointSlice

Service 需要采集：

```text
name
namespace
type
cluster_ip
ports
selector
external_name
annotations keys
```

Endpoint / EndpointSlice 需要采集：

```text
service_name
addresses
ports
ready
serving
terminating
targetRef kind
targetRef name
```

需要识别的问题：

```text
Service selector 不匹配
Endpoint 为空
Endpoint 指向的 Pod 未 Ready
Service targetPort 配置错误
Ingress 后端 Service 无可用 Endpoint
```

---

#### 29.4.6 Ingress 信息

需要采集：

```text
name
namespace
ingress_class_name
rules.host
rules.path
backend service name
backend service port
tls hosts
annotations keys
```

需要识别的问题：

```text
Ingress backend Service 不存在
Ingress backend Service 没有 Endpoint
路径规则不匹配
IngressClass 配置异常
Nginx Ingress 返回 502
Nginx Ingress 返回 503
Nginx Ingress 返回 504
Nginx Ingress 返回 499
```

---

#### 29.4.7 HPA 信息

需要采集：

```text
name
namespace
scale_target_ref
min_replicas
max_replicas
current_replicas
desired_replicas
current_metrics
target_metrics
conditions
```

需要识别的问题：

```text
指标不可用
未按预期扩容
扩容频繁抖动
currentReplicas 与 desiredReplicas 长时间不一致
CPU / 内存指标获取失败
```

---

#### 29.4.8 PVC / PV 信息

PVC 需要采集：

```text
name
namespace
phase
storage_class_name
access_modes
requested_storage
volume_name
conditions
```

可选 PV 需要采集：

```text
name
phase
capacity
access_modes
reclaim_policy
storage_class_name
claim_ref
```

需要识别的问题：

```text
PVC Pending
PV 不存在
StorageClass 不存在
FailedMount
FailedAttachVolume
挂载超时
```

---

#### 29.4.9 Node 信息，可选

如果需要分析 NodeNotReady、Pod Evicted、DiskPressure、MemoryPressure、Pod Pending 等问题，需要采集 Node。

Node 需要采集：

```text
name
conditions
capacity
allocatable
taints
labels
podCIDR
providerID
```

重点关注：

```text
Ready
MemoryPressure
DiskPressure
PIDPressure
NetworkUnavailable
```

Node 是集群级资源，第一阶段可以作为可选能力，单独配置 ClusterRole 只读权限。

---

#### 29.4.10 Prometheus 指标，可选

如果系统已有 Prometheus，可选接入以下指标：

```text
Pod CPU 使用率
Pod Memory 使用率
Container restart count
Pod 网络 RX/TX
Node CPU 使用率
Node Memory 使用率
Node Disk 使用率
Ingress request count
Ingress 4xx / 5xx / 499
Ingress latency
HPA current metrics
```

Prometheus 指标采集不是 Kubernetes API 权限的一部分，需要单独配置 Prometheus 数据源。

---

### 29.5 K8s 数据库设计

#### 29.5.1 Kubernetes 集群配置表

```sql
CREATE TABLE k8s_cluster (
    id BIGSERIAL PRIMARY KEY,

    name VARCHAR(120) NOT NULL,
    environment VARCHAR(50),
    api_server TEXT NOT NULL,

    auth_type VARCHAR(50) NOT NULL,
    kubeconfig_ref TEXT,
    bearer_token_ref TEXT,
    ca_cert_ref TEXT,
    client_cert_ref TEXT,
    client_key_ref TEXT,

    allowed_namespaces JSONB,
    enabled BOOLEAN DEFAULT true,

    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
name                 集群名称，例如：prod-k8s-01
environment          环境，例如：prod、test、dr
api_server           Kubernetes API Server 地址
auth_type            认证方式：kubeconfig、bearer_token、in_cluster
kubeconfig_ref       加密后的 kubeconfig 引用
bearer_token_ref     加密后的 token 引用
ca_cert_ref          加密后的 CA 证书引用
client_cert_ref      加密后的客户端证书引用
client_key_ref       加密后的客户端私钥引用
allowed_namespaces   允许查询的 namespace 列表
enabled              是否启用
```

安全要求：

```text
1. kubeconfig、token、证书、私钥必须加密保存。
2. API 返回集群配置时不得返回明文 token、kubeconfig 或私钥。
3. allowed_namespaces 为空时，不允许默认访问所有 namespace。
4. 生产环境建议按 namespace 显式授权。
```

---

#### 29.5.2 K8s 诊断任务表

```sql
CREATE TABLE k8s_diagnosis_task (
    id BIGSERIAL PRIMARY KEY,

    cluster_id BIGINT REFERENCES k8s_cluster(id),

    diagnosis_type VARCHAR(50) NOT NULL,
    question TEXT,

    alert_name VARCHAR(200),
    severity VARCHAR(50),

    namespace VARCHAR(120),
    pod_name VARCHAR(255),
    container_name VARCHAR(255),
    workload_kind VARCHAR(50),
    workload_name VARCHAR(255),
    service_name VARCHAR(255),
    ingress_name VARCHAR(255),
    node_name VARCHAR(255),

    time_start TIMESTAMP,
    time_end TIMESTAMP,

    status VARCHAR(50) DEFAULT 'pending',
    error_message TEXT,

    k8s_context JSONB,
    log_samples JSONB,
    metric_samples JSONB,
    retrieved_chunks JSONB,
    result JSONB,

    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
diagnosis_type     alert、pod、ingress、service、node、pvc、hpa
k8s_context        本次采集到的 K8s 结构化上下文
log_samples        本次关联到的日志样本
metric_samples     本次关联到的指标样本
retrieved_chunks   本次分析引用的知识库片段
result             LLM 分析结果
```

---

### 29.6 后端目录结构扩展

在现有后端目录基础上增加：

```text
backend/
  internal/
    model/
      k8s_cluster.go
      k8s_diagnosis_task.go

    handler/
      k8s_cluster_handler.go
      k8s_diagnosis_handler.go

    service/
      k8s_cluster_service.go
      k8s_collector_service.go
      k8s_diagnosis_service.go
      k8s_context_builder.go

    repository/
      k8s_cluster_repository.go
      k8s_diagnosis_repository.go

    client/
      kubernetes_client.go
      prometheus_client.go

    dto/
      k8s_dto.go
```

---

### 29.7 K8s Client 要求

实现文件：

```text
backend/internal/client/kubernetes_client.go
```

需要提供方法：

```go
type K8sClusterConfig struct {
    ID                int64
    Name              string
    Environment       string
    APIServer         string
    AuthType          string
    Kubeconfig        string
    BearerToken       string
    CACert            string
    ClientCert        string
    ClientKey         string
    AllowedNamespaces []string
}

type K8sLogOptions struct {
    Namespace     string
    PodName       string
    ContainerName string
    TailLines     int64
    Previous      bool
    MaxBytes      int64
}

type KubernetesClient interface {
    Test(ctx context.Context, cfg K8sClusterConfig) error

    GetPod(ctx context.Context, cfg K8sClusterConfig, namespace string, podName string) (*PodInfo, error)
    ListPods(ctx context.Context, cfg K8sClusterConfig, namespace string, selector map[string]string) ([]PodInfo, error)
    GetPodEvents(ctx context.Context, cfg K8sClusterConfig, namespace string, podName string) ([]K8sEvent, error)
    GetPodLogs(ctx context.Context, cfg K8sClusterConfig, opts K8sLogOptions) ([]LogItem, error)

    GetDeployment(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*WorkloadInfo, error)
    GetStatefulSet(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*WorkloadInfo, error)
    GetDaemonSet(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*WorkloadInfo, error)
    GetReplicaSet(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*WorkloadInfo, error)

    GetService(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*ServiceInfo, error)
    GetEndpoints(ctx context.Context, cfg K8sClusterConfig, namespace string, serviceName string) (*EndpointInfo, error)
    GetIngress(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*IngressInfo, error)

    GetHPA(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*HPAInfo, error)
    GetPVC(ctx context.Context, cfg K8sClusterConfig, namespace string, name string) (*PVCInfo, error)

    GetNode(ctx context.Context, cfg K8sClusterConfig, nodeName string) (*NodeInfo, error)
}
```

实现要求：

```text
1. 使用 client-go 调用 Kubernetes API。
2. 不通过 shell 执行 kubectl。
3. 不允许执行 exec、attach、portforward。
4. 所有读取操作必须校验 namespace 是否在 allowed_namespaces 内。
5. 所有日志读取必须限制 tailLines 和 maxBytes。
6. 返回给 LLM 前必须脱敏。
7. 不返回 Secret 明文。
```

---

### 29.8 API 设计

#### 29.8.1 创建 K8s 集群配置

```http
POST /api/k8s/clusters
Content-Type: application/json
```

请求示例：

```json
{
  "name": "prod-k8s-01",
  "environment": "prod",
  "apiServer": "https://10.10.10.10:6443",
  "authType": "bearer_token",
  "bearerToken": "******",
  "caCert": "-----BEGIN CERTIFICATE-----...",
  "allowedNamespaces": ["pay", "loan", "core"]
}
```

说明：

```text
authType 支持 kubeconfig、bearer_token、in_cluster
kubeconfig、bearerToken、caCert、clientCert、clientKey 只在创建或更新时提交
接口返回时不返回明文凭据
```

---

#### 29.8.2 查询 K8s 集群列表

```http
GET /api/k8s/clusters
```

返回：

```json
{
  "items": [
    {
      "id": 1,
      "name": "prod-k8s-01",
      "environment": "prod",
      "apiServer": "https://10.10.10.10:6443",
      "authType": "bearer_token",
      "allowedNamespaces": ["pay", "loan", "core"],
      "enabled": true
    }
  ],
  "total": 1
}
```

---

#### 29.8.3 更新 K8s 集群配置

```http
PUT /api/k8s/clusters/{id}
Content-Type: application/json
```

---

#### 29.8.4 删除 K8s 集群配置

```http
DELETE /api/k8s/clusters/{id}
```

---

#### 29.8.5 测试 K8s 集群连接

```http
POST /api/k8s/clusters/{id}/test
```

返回：

```json
{
  "ok": true,
  "message": "连接成功"
}
```

---

#### 29.8.6 K8s 告警诊断

```http
POST /api/k8s/diagnosis/alert
Content-Type: application/json
```

请求：

```json
{
  "clusterId": 1,
  "alertName": "KubePodCrashLooping",
  "severity": "critical",
  "namespace": "pay",
  "podName": "pay-core-6f8b7d9c7d-xk2lm",
  "containerName": "pay-core",
  "question": "这个 Pod 为什么一直重启？",
  "timeStart": "2026-07-07T09:30:00+08:00",
  "timeEnd": "2026-07-07T10:00:00+08:00",
  "topK": 5
}
```

返回：

```json
{
  "taskId": 1001,
  "status": "success",
  "summary": "pay-core Pod 处于 CrashLoopBackOff，最近 10 分钟多次重启。",
  "riskLevel": "high",
  "evidence": [
    "Pod containerStatuses 中 restartCount 持续增加",
    "previous logs 中存在数据库连接超时异常",
    "Events 中存在 BackOff restarting failed container"
  ],
  "possibleCauses": [
    "应用启动阶段依赖数据库超时",
    "配置中数据库连接地址或连接池参数异常",
    "近期发布后连接初始化逻辑异常"
  ],
  "suggestions": [
    "检查 previous logs 中首次异常堆栈",
    "检查数据库连接数、慢 SQL 和网络连通性",
    "确认最近一次发布或配置变更",
    "如需回滚或重启，必须按生产变更流程审批"
  ],
  "riskTips": [
    "不要直接删除 Pod 或重启 Deployment，需先确认影响范围并遵守变更流程。"
  ],
  "citations": [
    {
      "documentId": 12,
      "documentTitle": "K8s Pod 重启告警处置手册",
      "sourceSection": "3. CrashLoopBackOff 排查"
    }
  ]
}
```

---

#### 29.8.7 Pod 诊断

```http
POST /api/k8s/diagnosis/pod
Content-Type: application/json
```

请求：

```json
{
  "clusterId": 1,
  "namespace": "pay",
  "podName": "pay-core-6f8b7d9c7d-xk2lm",
  "containerName": "pay-core",
  "question": "分析这个 Pod 的异常原因",
  "includeLogs": true,
  "includePreviousLogs": true,
  "includeMetrics": true,
  "topK": 5
}
```

---

#### 29.8.8 Ingress 诊断

```http
POST /api/k8s/diagnosis/ingress
Content-Type: application/json
```

请求：

```json
{
  "clusterId": 1,
  "namespace": "pay",
  "ingressName": "pay-ingress",
  "question": "最近支付接口出现 502/504，帮忙分析可能原因",
  "includeNginxIngressLogs": true,
  "includeMetrics": true,
  "topK": 5
}
```

---

#### 29.8.9 Service 诊断

```http
POST /api/k8s/diagnosis/service
Content-Type: application/json
```

请求：

```json
{
  "clusterId": 1,
  "namespace": "pay",
  "serviceName": "pay-core-svc",
  "question": "这个服务为什么访问不通？",
  "topK": 5
}
```

---

#### 29.8.10 Node 诊断，可选

```http
POST /api/k8s/diagnosis/node
Content-Type: application/json
```

请求：

```json
{
  "clusterId": 1,
  "nodeName": "worker-01",
  "question": "这个节点为什么 NotReady？",
  "includeMetrics": true,
  "topK": 5
}
```

---

### 29.9 K8s 采集规则

#### 29.9.1 Namespace 限制

K8s 采集必须遵守：

```text
1. 每个集群必须配置 allowed_namespaces。
2. 用户请求中的 namespace 必须在 allowed_namespaces 内。
3. 不允许默认访问所有 namespace。
4. 不允许跨 namespace 自动扩大采集范围。
5. 如果告警中 namespace 为空，必须要求用户补充或根据白名单拒绝采集。
```

---

#### 29.9.2 日志读取限制

K8s Pod 日志读取必须遵守：

```text
1. 默认只读取最近 300 行。
2. 单次日志最大 256KB。
3. 支持读取 current logs 和 previous logs。
4. 不允许读取无限日志。
5. 不允许 watch 实时日志流进入 LLM。
6. 不允许把未脱敏日志发送给 LLM。
```

---

#### 29.9.3 Secret 和 ConfigMap 限制

```text
1. 默认不读取 Secret。
2. 不返回 Secret 明文。
3. 如果 Pod 引用了 Secret，只返回 Secret 名称和 key 名称。
4. 默认不读取 ConfigMap 内容。
5. 如果确需读取 ConfigMap 内容，必须单独配置权限，并进行敏感字段脱敏。
6. 禁止把 password、token、secret、access_key、private_key、authorization、cookie 等字段发送给 LLM。
```

---

#### 29.9.4 高风险操作限制

K8s 诊断助手不允许执行：

```text
kubectl exec
kubectl attach
kubectl port-forward
kubectl delete
kubectl apply
kubectl patch
kubectl edit
kubectl scale
kubectl rollout restart
kubectl drain
kubectl cordon
kubectl uncordon
```

LLM 可以给出排查建议，但涉及以下动作必须提示生产变更审批：

```text
重启 Pod
删除 Pod
扩容
缩容
回滚
修改 Deployment
修改 StatefulSet
修改 ConfigMap
修改 Ingress
修改 Service
迁移流量
节点隔离
节点驱逐
```

---

### 29.10 K8s RBAC 建议

#### 29.10.1 Namespace 级只读 Role

建议第一阶段按业务 namespace 授权，不直接给全集群权限。

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-ai-diagnosis
  namespace: ops-tools
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: k8s-ai-diagnosis-reader
  namespace: pay
rules:
  - apiGroups: [""]
    resources:
      - pods
      - pods/log
      - services
      - endpoints
      - events
      - persistentvolumeclaims
    verbs: ["get", "list", "watch"]

  - apiGroups: ["discovery.k8s.io"]
    resources:
      - endpointslices
    verbs: ["get", "list", "watch"]

  - apiGroups: ["apps"]
    resources:
      - deployments
      - statefulsets
      - daemonsets
      - replicasets
    verbs: ["get", "list", "watch"]

  - apiGroups: ["networking.k8s.io"]
    resources:
      - ingresses
    verbs: ["get", "list", "watch"]

  - apiGroups: ["autoscaling"]
    resources:
      - horizontalpodautoscalers
    verbs: ["get", "list", "watch"]

  - apiGroups: ["batch"]
    resources:
      - jobs
      - cronjobs
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: k8s-ai-diagnosis-reader-binding
  namespace: pay
subjects:
  - kind: ServiceAccount
    name: k8s-ai-diagnosis
    namespace: ops-tools
roleRef:
  kind: Role
  name: k8s-ai-diagnosis-reader
  apiGroup: rbac.authorization.k8s.io
```

如果需要读取多个业务 namespace，需要在每个 namespace 下创建对应 RoleBinding。

---

#### 29.10.2 可选 ClusterRole

如果需要读取 Node、PV、StorageClass 等集群级资源，再增加只读 ClusterRole。

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-ai-diagnosis-cluster-reader
rules:
  - apiGroups: [""]
    resources:
      - nodes
      - namespaces
      - persistentvolumes
    verbs: ["get", "list", "watch"]

  - apiGroups: ["storage.k8s.io"]
    resources:
      - storageclasses
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k8s-ai-diagnosis-cluster-reader-binding
subjects:
  - kind: ServiceAccount
    name: k8s-ai-diagnosis
    namespace: ops-tools
roleRef:
  kind: ClusterRole
  name: k8s-ai-diagnosis-cluster-reader
  apiGroup: rbac.authorization.k8s.io
```

---

#### 29.10.3 明确禁止授权

第一阶段不要授予以下权限：

```text
secrets
configmaps 明文读取
pods/exec
pods/attach
pods/portforward
serviceaccounts/token
create
update
patch
delete
deletecollection
```

---

### 29.11 K8s Prompt 设计

#### 29.11.1 K8s 告警分析 Prompt

```text
你是一个资深银行生产 Kubernetes 运维专家。

请基于【K8s 告警信息】、【K8s 采集上下文】、【日志样本】、【指标样本】和【知识库内容】分析用户问题。

要求：
1. 必须区分 K8s 采集到的事实、日志中可以直接观察到的事实、指标趋势、知识库依据、以及推测性的可能原因。
2. 不要编造不存在的 Pod、namespace、node、service、endpoint、日志、时间点或指标。
3. 如果知识库没有相关依据，请明确说明：“知识库中未找到明确依据”。
4. 涉及生产命令时，只能作为排查建议展示，不允许描述为可以直接执行。
5. 涉及重启、删除 Pod、扩容、缩容、回滚、修改配置、修改 Ingress、修改 Service、节点隔离、节点驱逐等高风险操作时，必须提示需要按生产变更流程审批。
6. 不要建议直接执行 kubectl exec、delete、patch、apply、scale、rollout restart 等操作。
7. 输出需要包含 K8s 证据、日志证据、指标证据和知识库引用。
8. 如果当前采集信息不足，请明确说明还缺少哪些信息。

用户问题：
{{question}}

告警信息：
{{alert}}

K8s 采集上下文：
{{k8s_context}}

日志样本：
{{log_samples}}

指标样本：
{{metric_samples}}

知识库内容：
{{chunks}}

请按以下格式回答：

## 异常摘要

## K8s 证据

## 日志证据

## 指标证据

## 可能原因

## 建议排查步骤

## 建议处理动作

## 风险提示

## 知识库引用
```

---

#### 29.11.2 Pod 诊断 Prompt

```text
你是一个资深银行生产 Kubernetes Pod 排查专家。

请基于以下 Pod 状态、Events、日志、Workload 配置、Service/Endpoint 信息和知识库内容，分析 Pod 异常原因。

要求：
1. 优先根据 Pod containerStatuses、lastState、restartCount、Events、previous logs 判断原因。
2. 对 CrashLoopBackOff，需要优先查看 previous logs。
3. 对 OOMKilled，需要结合内存 limit、lastState reason 和内存指标判断。
4. 对 ImagePullBackOff，需要结合 Events、image、imagePullSecrets 引用情况判断。
5. 对 Pending，需要结合 Events、调度失败原因、PVC、Node 资源判断。
6. 对 Readiness/Liveness probe failed，需要结合 probe 配置、Events、容器日志和 Endpoint 状态判断。
7. 不要编造采集信息中不存在的原因。
8. 涉及生产变更动作必须提示审批。

用户问题：
{{question}}

Pod 信息：
{{pod_info}}

Pod Events：
{{pod_events}}

当前日志：
{{current_logs}}

上一次容器日志：
{{previous_logs}}

Workload 信息：
{{workload_info}}

Service / Endpoint 信息：
{{service_endpoint_info}}

Node 信息：
{{node_info}}

指标样本：
{{metric_samples}}

知识库内容：
{{chunks}}

请按以下格式回答：

## 结论

## 关键证据

## 可能原因

## 排查步骤

## 建议处理动作

## 风险提示

## 知识库引用
```

---

#### 29.11.3 Ingress / Service 诊断 Prompt

```text
你是一个资深银行生产 Kubernetes 入口流量排查专家。

请基于 Ingress、Service、Endpoint、Pod Ready 状态、Nginx Ingress 日志、指标样本和知识库内容，分析访问异常原因。

重点关注：
1. Ingress 规则是否正确指向 Service。
2. Service selector 是否能匹配到 Pod。
3. Endpoint 是否为空。
4. Endpoint 对应 Pod 是否 Ready。
5. targetPort 是否和容器端口一致。
6. Nginx Ingress 是否存在 502、503、504、499、upstream timeout、no endpoints available 等日志。
7. 不要编造不存在的 host、path、service、endpoint 或日志。

用户问题：
{{question}}

Ingress 信息：
{{ingress_info}}

Service 信息：
{{service_info}}

Endpoint 信息：
{{endpoint_info}}

后端 Pod 信息：
{{backend_pods}}

Nginx Ingress 日志：
{{ingress_logs}}

指标样本：
{{metric_samples}}

知识库内容：
{{chunks}}

请按以下格式回答：

## 异常摘要

## 访问链路证据

## 日志证据

## 可能原因

## 建议排查步骤

## 风险提示

## 知识库引用
```

---

### 29.12 LogAnalysisService 与 K8sDiagnosisService 关系

现有 LogAnalysisService 继续负责 Elasticsearch 和服务器日志文件分析。

新增 K8sDiagnosisService 负责 K8s 上下文采集和 K8s 告警诊断。

二者关系：

```text
K8sDiagnosisService
    ↓
K8sCollectorService 采集 Kubernetes 上下文
    ↓
可选调用 LogAnalysisService 获取关联日志样本
    ↓
可选调用 PrometheusClient 获取指标样本
    ↓
调用 RAGService 检索知识库
    ↓
组装 K8s Prompt
    ↓
调用默认 LLM
    ↓
保存 k8s_diagnosis_task
```

---

### 29.13 前端页面扩展

#### 29.13.1 K8sClusterPage

功能：

```text
1. 查看 K8s 集群列表
2. 新增 K8s 集群配置
3. 支持 kubeconfig、bearer_token、in_cluster 三种认证方式
4. 配置 allowed_namespaces
5. 测试集群连接
6. 启用或禁用集群
7. 不展示明文 token、kubeconfig、证书或私钥
```

---

#### 29.13.2 K8sDiagnosisPage

功能：

```text
1. 选择集群
2. 选择诊断类型：告警、Pod、Ingress、Service、Node
3. 输入 namespace、pod、container、service、ingress、node 等参数
4. 可粘贴 Alertmanager 告警 JSON
5. 可选择是否采集日志
6. 可选择是否采集 previous logs
7. 可选择是否关联 Prometheus 指标
8. 提交诊断
9. 展示异常摘要、关键证据、可能原因、建议排查步骤、风险提示
10. 展示 K8s 采集上下文摘要
11. 展示日志证据
12. 展示知识库引用
13. 提示 AI 分析仅供排查参考
```

---

#### 29.13.3 前端 API 扩展

新增文件：

```text
frontend/src/api/k8sApi.ts
```

需要实现：

```ts
export async function listK8sClusters()

export async function createK8sCluster(data: {
  name: string
  environment?: string
  apiServer: string
  authType: 'kubeconfig' | 'bearer_token' | 'in_cluster'
  kubeconfig?: string
  bearerToken?: string
  caCert?: string
  clientCert?: string
  clientKey?: string
  allowedNamespaces: string[]
  enabled?: boolean
})

export async function updateK8sCluster(id: number, data: Partial<K8sClusterInput>)

export async function deleteK8sCluster(id: number)

export async function testK8sCluster(id: number)

export async function diagnoseK8sAlert(data: {
  clusterId: number
  alertName?: string
  severity?: string
  namespace: string
  podName?: string
  containerName?: string
  workloadKind?: string
  workloadName?: string
  serviceName?: string
  ingressName?: string
  nodeName?: string
  question?: string
  timeStart?: string
  timeEnd?: string
  topK?: number
})

export async function diagnoseK8sPod(data: {
  clusterId: number
  namespace: string
  podName: string
  containerName?: string
  question?: string
  includeLogs?: boolean
  includePreviousLogs?: boolean
  includeMetrics?: boolean
  topK?: number
})

export async function diagnoseK8sIngress(data: {
  clusterId: number
  namespace: string
  ingressName: string
  question?: string
  includeNginxIngressLogs?: boolean
  includeMetrics?: boolean
  topK?: number
})

export async function diagnoseK8sService(data: {
  clusterId: number
  namespace: string
  serviceName: string
  question?: string
  topK?: number
})

export async function diagnoseK8sNode(data: {
  clusterId: number
  nodeName: string
  question?: string
  includeMetrics?: boolean
  topK?: number
})
```

---

### 29.14 环境变量扩展

```env
K8S_COLLECT_TIMEOUT_SECONDS=15
K8S_LOG_TAIL_LINES=300
K8S_LOG_MAX_BYTES=262144
K8S_LOG_PREVIOUS_ENABLED=true
K8S_ALLOWED_ALL_NAMESPACES=false

PROMETHEUS_ENABLED=false
PROMETHEUS_BASE_URL=http://prometheus.internal.local:9090
PROMETHEUS_QUERY_TIMEOUT_SECONDS=10
```

说明：

```text
K8S_ALLOWED_ALL_NAMESPACES 默认 false。
生产环境不允许默认查询所有 namespace。
如果需要跨 namespace 查询，必须在集群配置 allowed_namespaces 中显式配置。
```

---

### 29.15 安全要求扩展

在原有安全要求基础上增加：

```text
1. K8s 集群凭据必须加密保存。
2. API 返回 K8s 集群配置时不得返回明文 token、kubeconfig、证书或私钥。
3. K8s 查询必须限制在 allowed_namespaces 内。
4. 不允许默认访问所有 namespace。
5. 不允许读取 Secret 明文。
6. 默认不读取 ConfigMap 内容。
7. 不允许调用 exec、attach、port-forward。
8. 不允许 create、update、patch、delete Kubernetes 资源。
9. 不允许自动执行 kubectl 命令。
10. Pod 日志进入 LLM 前必须脱敏。
11. Pod 日志读取必须限制行数和字节数。
12. K8s 诊断结果只能作为排查建议，不允许自动执行修复动作。
13. 涉及删除、重启、扩容、缩容、回滚、修改配置、修改网络规则、节点隔离、节点驱逐等动作，必须提示生产变更审批。
14. 后端需要记录 K8s 诊断任务和返回结果，方便审计。
```

---

### 29.16 Codex 开发任务扩展

### Task 17：K8s 数据库模型和迁移

目标：

```text
创建 k8s_cluster、k8s_diagnosis_task 表。
```

验收标准：

```text
1. migrations 中包含 K8s 表结构。
2. GORM model 与表结构对应。
3. k8s_cluster 可以保存集群基础配置。
4. K8s 凭据字段只保存加密引用，不保存明文。
```

---

### Task 18：K8s 集群配置管理

目标：

```text
实现 K8s 集群配置的新增、查询、更新、删除、测试连接。
```

验收标准：

```text
1. 可以创建 K8s 集群配置。
2. 支持 kubeconfig、bearer_token、in_cluster 认证方式。
3. token、kubeconfig、证书、私钥加密保存。
4. 查询接口不返回明文凭据。
5. 可以配置 allowed_namespaces。
6. 可以测试 Kubernetes API 连接。
```

---

### Task 19：KubernetesClient 实现

目标：

```text
使用 client-go 实现 Kubernetes 只读采集客户端。
```

验收标准：

```text
1. 可以读取 Pod 状态。
2. 可以读取 Pod Events。
3. 可以读取 Pod 当前日志。
4. 可以读取 Pod previous 日志。
5. 可以读取 Deployment、StatefulSet、DaemonSet、ReplicaSet。
6. 可以读取 Service、Endpoint、EndpointSlice。
7. 可以读取 Ingress、HPA、PVC。
8. 可选读取 Node。
9. 不通过 shell 执行 kubectl。
10. 不实现 exec、attach、port-forward、delete、patch、apply。
```

---

### Task 20：K8s 采集上下文构造

目标：

```text
实现 K8sCollectorService 和 K8sContextBuilder。
```

验收标准：

```text
1. 输入 clusterId、namespace、podName 后，可以构造 Pod 诊断上下文。
2. 输入 ingressName 后，可以构造 Ingress 访问链路上下文。
3. 输入 serviceName 后，可以构造 Service / Endpoint 上下文。
4. 采集前校验 namespace 是否在 allowed_namespaces 内。
5. 日志采集有行数和字节数限制。
6. 进入 LLM 前完成脱敏。
```

---

### Task 21：K8s 告警诊断

目标：

```text
实现 /api/k8s/diagnosis/alert。
```

验收标准：

```text
1. 可以接收 Alertmanager 告警字段。
2. 可以根据 alertName 判断采集策略。
3. 可以自动补充 Pod、Events、Logs、Workload、Service、Endpoint 等上下文。
4. 可以检索相关 published 知识库文档。
5. 可以组装 K8s 告警分析 Prompt。
6. 可以调用默认 LLM 生成诊断结果。
7. 可以保存 k8s_diagnosis_task。
```

---

### Task 22：K8s Pod 诊断

目标：

```text
实现 /api/k8s/diagnosis/pod。
```

验收标准：

```text
1. 可以输入 namespace、podName、containerName。
2. 可以读取 Pod 状态、Events、当前日志和 previous 日志。
3. 可以反查所属 Workload。
4. 可以关联 Service / Endpoint。
5. 可以调用 LLM 输出异常摘要、关键证据、可能原因、建议排查步骤和风险提示。
```

---

### Task 23：K8s Ingress / Service 诊断

目标：

```text
实现 /api/k8s/diagnosis/ingress 和 /api/k8s/diagnosis/service。
```

验收标准：

```text
1. Ingress 诊断可以采集 Ingress、Service、Endpoint、后端 Pod。
2. Service 诊断可以采集 Service selector、Endpoint、Pod Ready 状态。
3. 可以识别 Endpoint 为空、selector 不匹配、Pod 未 Ready、targetPort 异常等问题。
4. 可以调用 LLM 生成访问链路分析。
```

---

### Task 24：K8s 前端页面

目标：

```text
实现 K8sClusterPage 和 K8sDiagnosisPage。
```

验收标准：

```text
1. 可以管理 K8s 集群配置。
2. 可以配置 allowed_namespaces。
3. 可以测试集群连接。
4. 可以提交 Pod 诊断。
5. 可以提交告警诊断。
6. 可以提交 Ingress / Service 诊断。
7. 可以展示异常摘要、关键证据、可能原因、排查建议、风险提示、知识库引用。
8. 页面提示 AI 分析仅供排查参考。
```

---

### 29.17 第一阶段 K8s MVP 范围

### 必须实现

```text
1. K8s 集群配置
2. K8s 集群连接测试
3. allowed_namespaces 控制
4. Pod 状态采集
5. Pod Events 采集
6. Pod 当前日志采集
7. Pod previous 日志采集
8. Deployment / StatefulSet / ReplicaSet 采集
9. Service / Endpoint 采集
10. K8s Pod 诊断接口
11. K8s 告警诊断接口
12. K8s Prompt
13. K8s 诊断结果保存
14. 前端 K8s 集群配置页面
15. 前端 K8s 诊断页面
```

### 可以暂缓

```text
1. Node 诊断
2. PV / StorageClass 诊断
3. HPA 诊断
4. Prometheus 指标接入
5. Nginx Ingress Controller 日志自动关联
6. Alertmanager Webhook 自动接入
7. 多集群权限管理
8. 与工单系统联动
9. 诊断报告导出
10. 实时事件流监听
```

---

### 29.18 K8s 示例告警测试

可以使用以下 Alertmanager 告警测试：

```json
{
  "alertName": "KubePodCrashLooping",
  "severity": "critical",
  "cluster": "prod-k8s-01",
  "namespace": "pay",
  "pod": "pay-core-6f8b7d9c7d-xk2lm",
  "container": "pay-core",
  "summary": "Pod is crash looping",
  "description": "Pod pay/pay-core-6f8b7d9c7d-xk2lm is restarting frequently"
}
```

期望回答应该包含：

```text
1. Pod 当前状态
2. restartCount
3. lastState reason
4. Events 中的 BackOff 或异常原因
5. previous logs 中的关键错误
6. 可能原因
7. 建议排查步骤
8. 不建议直接删除 Pod
9. 如需重启、回滚、扩容，必须走生产变更审批
10. 知识库引用
```

---

## 30. 简化用户管理设计

### 30.1 扩展目标

在现有运维知识库 RAG 系统基础上，增加简化用户管理能力。

系统需要支持用户登录，并区分两类角色：

```text
admin       管理员
user        普通用户
```

本阶段不做复杂多租户、不做细粒度系统权限、不做部门级权限模型，只实现最小可用的角色权限控制。

核心目标：

```text
1. 用户必须登录后才能使用系统。
2. admin 可以管理普通用户。
3. admin 可以启用、禁用、重置普通用户密码。
4. 普通用户只能使用知识库问答、查看自己创建的会话、查看自己提交的分析记录。
5. admin 可以查看和管理全局数据。
6. 所有问答、日志分析、文档上传、审核、模型配置、日志源配置都需要记录操作用户。
7. 系统需要基于登录用户维护上下文。
```

---

### 30.2 角色权限边界

#### admin 权限

admin 可以执行：

```text
1. 登录系统
2. 创建普通用户
3. 修改普通用户信息
4. 启用 / 禁用普通用户
5. 重置普通用户密码
6. 查看所有用户
7. 上传文档
8. 审核文档
9. 发布文档
10. 废弃文档
11. 删除文档
12. 配置 LLM 接口
13. 配置 Elasticsearch 日志源
14. 配置服务器日志源
15. 配置 K8s 集群
16. 查看所有问答记录
17. 查看所有日志分析任务
18. 查看所有 K8s 诊断任务
19. 查看系统审计日志
20. 管理所有会话上下文
```

#### 普通用户权限

普通用户可以执行：

```text
1. 登录系统
2. 修改自己的密码
3. 查看已发布知识库文档
4. 发起知识库问答
5. 创建自己的聊天会话
6. 查看自己的聊天历史
7. 删除自己的聊天会话
8. 发起日志分析
9. 查看自己的日志分析任务
10. 发起 K8s 诊断
11. 查看自己的 K8s 诊断任务
12. 上传文档草稿或待审核文档
```

普通用户不允许执行：

```text
1. 管理其他用户
2. 审核文档
3. 发布文档
4. 配置 LLM 接口
5. 配置日志源
6. 配置 K8s 集群
7. 查看其他用户的问答记录
8. 查看其他用户的会话上下文
9. 查看系统审计日志
10. 启用、禁用、删除其他用户
```

---

## 31. 用户认证设计

### 31.1 登录方式

第一阶段使用账号密码登录。

```text
1. 用户使用 username + password 登录。
2. 后端校验密码。
3. 登录成功后返回 JWT access_token。
4. 前端将 token 保存到 localStorage 或 sessionStorage。
5. 后续请求通过 Authorization: Bearer <token> 访问接口。
```

密码安全要求：

```text
1. 数据库只保存 password_hash，不保存明文密码。
2. 密码使用 bcrypt 哈希。
3. 创建用户时需要设置初始密码。
4. 普通用户首次登录后可以修改自己的密码。
5. admin 可以重置普通用户密码。
6. 禁用用户不能登录。
```

### 31.2 JWT 内容

JWT Payload 至少包含：

```json
{
  "userId": 1,
  "username": "admin",
  "role": "admin",
  "exp": 1780000000
}
```

后端中间件需要解析 JWT，并将当前用户信息注入 Gin Context。

建议注入字段：

```text
current_user_id
current_username
current_user_role
```

---

## 32. 用户数据库设计

### 32.1 用户表

```sql
CREATE TABLE app_user (
    id BIGSERIAL PRIMARY KEY,

    username VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(120),
    password_hash TEXT NOT NULL,

    role VARCHAR(30) NOT NULL DEFAULT 'user',
    enabled BOOLEAN NOT NULL DEFAULT true,

    last_login_at TIMESTAMP,
    password_updated_at TIMESTAMP,

    created_by BIGINT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),

    CONSTRAINT chk_app_user_role CHECK (role IN ('admin', 'user'))
);
```

字段说明：

```text
username             登录账号，唯一
display_name         显示名称
password_hash        bcrypt 哈希后的密码
role                 admin 或 user
enabled              是否启用
last_login_at        最近登录时间
password_updated_at  最近修改密码时间
created_by           创建该用户的管理员 ID
```

---

### 32.2 登录审计表

```sql
CREATE TABLE login_audit (
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT REFERENCES app_user(id),
    username VARCHAR(100),

    success BOOLEAN NOT NULL,
    failure_reason TEXT,

    client_ip VARCHAR(100),
    user_agent TEXT,

    created_at TIMESTAMP DEFAULT now()
);
```

用途：

```text
1. 记录登录成功和失败。
2. 方便后续排查异常登录。
3. 为后续安全审计预留基础。
```

---

### 32.3 初始化管理员

系统启动时需要支持初始化 admin 用户。

环境变量：

```env
INIT_ADMIN_USERNAME=admin
INIT_ADMIN_PASSWORD=Admin@123456
JWT_SECRET=change-me-jwt-secret
JWT_EXPIRE_HOURS=24
```

初始化规则：

```text
1. 如果 app_user 表中不存在任何 admin，则自动创建默认 admin。
2. 如果已经存在 admin，则不重复创建。
3. 默认密码只能用于开发和首次部署，生产环境必须修改。
4. 后端启动日志中不得打印明文密码。
```

---

## 33. 上下文维护设计

### 33.1 设计目标

系统需要基于登录用户维护上下文，但不能只按 user_id 保存全部上下文。

推荐分层：

```text
user_id
  └── conversation_id
        └── task_id
              └── tool_call_id / evidence_id
```

各层含义：

```text
user_id             表示这个上下文属于哪个登录用户
conversation_id     表示一次聊天会话
task_id             表示一次具体问答、日志分析、K8s 诊断或排障任务
tool_call_id        表示一次工具调用或外部数据采集
evidence_id         表示一次证据记录，例如日志样本、知识库片段、K8s 资源快照
```

不要只用 user_id 作为上下文 key。

错误示例：

```text
context_key = user_id
```

正确示例：

```text
context_key = user_id + conversation_id
context_key = user_id + conversation_id + task_id
```

这样可以避免同一个用户同时排查多个问题时互相污染上下文。

---

### 33.2 上下文类型

系统至少维护以下上下文：

```text
1. 用户上下文 user_context
2. 会话上下文 conversation_context
3. 最近消息 recent_messages
4. 会话摘要 conversation_summary
5. 当前任务状态 task_state
6. 检索上下文 retrieved_context
7. 工具结果上下文 tool_result_context
8. 审计上下文 audit_context
```

### 33.3 用户上下文 user_context

用户上下文来自登录用户，不由 LLM 生成。

示例：

```json
{
  "userId": 1,
  "username": "zhangsan",
  "role": "user",
  "enabled": true
}
```

用途：

```text
1. 权限判断
2. 数据归属
3. 审计记录
4. Prompt 中标识当前用户角色
```

注意：

```text
普通用户的 user_context 只用于权限和归属，不要把敏感信息传给 LLM。
```

---

### 33.4 会话上下文 conversation_context

一次聊天窗口对应一个 conversation。

用户可以有多个 conversation。

例如：

```text
会话 A：Redis 内存告警问答
会话 B：支付系统日志超时分析
会话 C：K8s Pod CrashLoopBackOff 诊断
```

后端需要根据 conversation_id 取对应上下文，不能混用。

---

### 33.5 任务状态 task_state

task_state 用于记录当前会话内正在排查的问题状态。

适用场景：

```text
1. 知识库连续问答
2. 日志分析
3. K8s Pod 诊断
4. K8s 告警诊断
5. SQL 慢查询排查
6. 数据库异常诊断
```

通用结构：

```json
{
  "taskType": "qa | log_analysis | k8s_diagnosis | db_diagnosis",
  "systemName": "支付系统",
  "componentName": "Redis",
  "environment": "prod",
  "problemSummary": "用户正在排查 Redis 内存告警",
  "knownFacts": [
    "知识库命中了 Redis 内存告警处置手册",
    "用户关注生产环境处理步骤"
  ],
  "checkedItems": [
    "知识库检索",
    "风险提示"
  ],
  "currentConclusion": "需要先查看 info memory、bigkeys 和 slowlog，不建议直接删除 key",
  "pendingQuestions": [
    "Redis 实例地址和端口未提供",
    "当前内存使用率未提供"
  ],
  "riskLevel": "medium",
  "nextActions": [
    "确认 Redis 内存指标",
    "查看 bigkey",
    "查看 slowlog"
  ]
}
```

---

## 34. 上下文数据库设计

### 34.1 会话表

```sql
CREATE TABLE conversation (
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT NOT NULL REFERENCES app_user(id),
    title VARCHAR(255),
    conversation_type VARCHAR(50) DEFAULT 'qa',

    system_name VARCHAR(100),
    component_name VARCHAR(100),
    environment VARCHAR(50),

    last_message_at TIMESTAMP,
    archived BOOLEAN DEFAULT false,

    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
conversation_type    qa、log_analysis、k8s_diagnosis、db_diagnosis
title                会话标题，可由 LLM 根据首轮问题生成
archived             用户是否归档该会话
```

权限规则：

```text
1. 普通用户只能访问自己的 conversation。
2. admin 可以访问所有 conversation。
```

---

### 34.2 会话消息表

```sql
CREATE TABLE conversation_message (
    id BIGSERIAL PRIMARY KEY,

    conversation_id BIGINT NOT NULL REFERENCES conversation(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES app_user(id),

    role VARCHAR(30) NOT NULL,
    content TEXT NOT NULL,

    model_name VARCHAR(100),
    token_count INT DEFAULT 0,

    retrieved_chunks JSONB,
    citations JSONB,
    tool_results JSONB,

    created_at TIMESTAMP DEFAULT now(),

    CONSTRAINT chk_conversation_message_role CHECK (role IN ('user', 'assistant', 'system', 'tool'))
);
```

用途：

```text
1. 保存原始多轮对话。
2. 支持后续上下文拼装。
3. 支持审计和追溯。
```

---

### 34.3 会话摘要表

```sql
CREATE TABLE conversation_summary (
    id BIGSERIAL PRIMARY KEY,

    conversation_id BIGINT NOT NULL UNIQUE REFERENCES conversation(id) ON DELETE CASCADE,

    summary TEXT,
    key_facts JSONB,
    user_constraints JSONB,
    current_topic TEXT,

    last_message_id BIGINT,
    version INT DEFAULT 1,

    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
summary              会话摘要
key_facts            关键事实，例如系统、组件、报错、结论
user_constraints     用户明确约束，例如“不要执行命令”“只看知识库依据”
current_topic        当前话题
last_message_id      摘要覆盖到哪条消息
version              摘要版本
```

---

### 34.4 任务状态表

```sql
CREATE TABLE task_state (
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT NOT NULL REFERENCES app_user(id),
    conversation_id BIGINT NOT NULL REFERENCES conversation(id) ON DELETE CASCADE,

    task_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',

    system_name VARCHAR(100),
    component_name VARCHAR(100),
    environment VARCHAR(50),

    state JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

字段说明：

```text
task_type       qa、log_analysis、k8s_diagnosis、db_diagnosis
status          active、finished、archived
state           结构化任务状态 JSON
```

---

### 34.5 工具调用记录表

```sql
CREATE TABLE tool_call_record (
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT REFERENCES app_user(id),
    conversation_id BIGINT REFERENCES conversation(id),
    task_state_id BIGINT REFERENCES task_state(id),

    tool_name VARCHAR(100) NOT NULL,
    tool_type VARCHAR(50),

    request_summary JSONB,
    result_summary JSONB,
    raw_result_ref TEXT,

    success BOOLEAN DEFAULT true,
    error_message TEXT,

    created_at TIMESTAMP DEFAULT now()
);
```

用途：

```text
1. 记录知识库检索、日志读取、K8s 采集、LLM 调用等工具行为。
2. 原始大结果不直接塞入上下文，只保存引用路径或摘要。
3. 便于审计和问题复盘。
```

---

### 34.6 上下文快照表

```sql
CREATE TABLE context_snapshot (
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT REFERENCES app_user(id),
    conversation_id BIGINT REFERENCES conversation(id),
    task_state_id BIGINT REFERENCES task_state(id),

    snapshot_type VARCHAR(50) NOT NULL,
    prompt_context JSONB NOT NULL,

    model_name VARCHAR(100),
    token_count INT DEFAULT 0,

    created_at TIMESTAMP DEFAULT now()
);
```

用途：

```text
1. 保存每次调用 LLM 前实际使用的上下文。
2. 方便审计模型回答依据。
3. 方便排查回答质量问题。
```

注意：

```text
1. snapshot 中不得保存密码、token、私钥、连接串等敏感信息。
2. 日志样本进入 snapshot 前必须脱敏。
3. 大量日志、K8s 原始资源、ES 原始结果不建议完整保存，只保存摘要和引用。
```

---

## 35. 现有表结构调整

### 35.1 kb_document 调整

将：

```sql
created_by VARCHAR(100),
reviewed_by VARCHAR(100),
```

调整为：

```sql
created_by BIGINT REFERENCES app_user(id),
reviewed_by BIGINT REFERENCES app_user(id),
```

---

### 35.2 qa_record 调整

原 qa_record 需要增加 user_id 和 conversation_id。

```sql
ALTER TABLE qa_record
ADD COLUMN user_id BIGINT REFERENCES app_user(id),
ADD COLUMN conversation_id BIGINT REFERENCES conversation(id);
```

建议最终结构：

```sql
CREATE TABLE qa_record (
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT REFERENCES app_user(id),
    conversation_id BIGINT REFERENCES conversation(id),

    question TEXT NOT NULL,
    answer TEXT NOT NULL,

    retrieved_chunks JSONB,
    model_name VARCHAR(100),

    created_at TIMESTAMP DEFAULT now()
);
```

---

### 35.3 log_analysis_task 调整

增加：

```sql
ALTER TABLE log_analysis_task
ADD COLUMN user_id BIGINT REFERENCES app_user(id),
ADD COLUMN conversation_id BIGINT REFERENCES conversation(id),
ADD COLUMN task_state_id BIGINT REFERENCES task_state(id);
```

原来的 `created_by VARCHAR(100)` 可以改为：

```sql
created_by BIGINT REFERENCES app_user(id)
```

---

### 35.4 log_source 调整

原来的 `created_by VARCHAR(100)` 改为：

```sql
created_by BIGINT REFERENCES app_user(id)
```

权限要求：

```text
1. 只有 admin 可以创建、修改、删除日志源。
2. 普通用户只能选择已启用的日志源进行分析。
3. 普通用户不能看到日志源凭据字段。
```

---

### 35.5 k8s_cluster 和 k8s_diagnosis_task 调整

如果系统实现 K8s 扩展，需要同步调整：

```sql
ALTER TABLE k8s_cluster
ADD COLUMN created_by BIGINT REFERENCES app_user(id);

ALTER TABLE k8s_diagnosis_task
ADD COLUMN user_id BIGINT REFERENCES app_user(id),
ADD COLUMN conversation_id BIGINT REFERENCES conversation(id),
ADD COLUMN task_state_id BIGINT REFERENCES task_state(id);
```

权限要求：

```text
1. 只有 admin 可以配置 K8s 集群。
2. 普通用户只能使用已启用的 K8s 集群做只读诊断。
3. 普通用户只能查看自己的 K8s 诊断任务。
4. admin 可以查看所有诊断任务。
```

---

## 36. API 扩展设计

### 36.1 登录

```http
POST /api/auth/login
Content-Type: application/json
```

请求：

```json
{
  "username": "admin",
  "password": "Admin@123456"
}
```

返回：

```json
{
  "accessToken": "jwt-token",
  "user": {
    "id": 1,
    "username": "admin",
    "displayName": "系统管理员",
    "role": "admin"
  }
}
```

---

### 36.2 当前用户信息

```http
GET /api/auth/me
Authorization: Bearer <token>
```

返回：

```json
{
  "id": 1,
  "username": "admin",
  "displayName": "系统管理员",
  "role": "admin",
  "enabled": true
}
```

---

### 36.3 修改自己的密码

```http
POST /api/auth/change-password
Authorization: Bearer <token>
Content-Type: application/json
```

请求：

```json
{
  "oldPassword": "old-password",
  "newPassword": "new-password"
}
```

---

### 36.4 管理员创建用户

```http
POST /api/users
Authorization: Bearer <admin-token>
Content-Type: application/json
```

请求：

```json
{
  "username": "zhangsan",
  "displayName": "张三",
  "password": "User@123456",
  "role": "user",
  "enabled": true
}
```

---

### 36.5 管理员查询用户列表

```http
GET /api/users?page=1&pageSize=10&keyword=zhang
Authorization: Bearer <admin-token>
```

---

### 36.6 管理员更新用户

```http
PUT /api/users/{id}
Authorization: Bearer <admin-token>
Content-Type: application/json
```

请求：

```json
{
  "displayName": "张三",
  "role": "user",
  "enabled": true
}
```

限制：

```text
1. 普通用户不能调用该接口。
2. admin 不能把最后一个 admin 禁用。
3. admin 不能把最后一个 admin 改成普通用户。
```

---

### 36.7 管理员重置用户密码

```http
POST /api/users/{id}/reset-password
Authorization: Bearer <admin-token>
Content-Type: application/json
```

请求：

```json
{
  "newPassword": "NewPassword@123"
}
```

---

### 36.8 会话列表

```http
GET /api/conversations
Authorization: Bearer <token>
```

普通用户只返回自己的会话。

admin 可以通过参数查看全部：

```http
GET /api/conversations?all=true
Authorization: Bearer <admin-token>
```

---

### 36.9 创建会话

```http
POST /api/conversations
Authorization: Bearer <token>
Content-Type: application/json
```

请求：

```json
{
  "title": "Redis 内存告警排查",
  "conversationType": "qa",
  "systemName": "支付系统",
  "componentName": "Redis",
  "environment": "prod"
}
```

---

### 36.10 查看会话消息

```http
GET /api/conversations/{id}/messages
Authorization: Bearer <token>
```

权限要求：

```text
1. 普通用户只能查看自己的会话消息。
2. admin 可以查看所有会话消息。
```

---

### 36.11 删除或归档会话

```http
DELETE /api/conversations/{id}
Authorization: Bearer <token>
```

第一阶段建议软删除：

```text
archived = true
```

---

### 36.12 知识库问答接口调整

原接口：

```http
POST /api/qa/ask
```

请求增加 conversationId：

```json
{
  "conversationId": 1001,
  "question": "Redis 内存告警怎么处理？",
  "systemName": "支付系统",
  "componentName": "Redis",
  "topK": 5
}
```

处理规则：

```text
1. 必须登录。
2. 如果没有传 conversationId，后端自动创建一个新的 conversation。
3. 如果传了 conversationId，需要校验该会话是否属于当前用户。
4. admin 可以访问任意 conversation。
5. 本轮用户问题写入 conversation_message。
6. 系统读取最近消息、会话摘要、task_state、知识库检索结果后组装 Prompt。
7. LLM 回答写入 conversation_message。
8. 更新 qa_record。
9. 更新 conversation_summary。
10. 更新 task_state。
11. 保存 context_snapshot。
```

返回增加 conversationId 和 messageId：

```json
{
  "conversationId": 1001,
  "messageId": 9002,
  "answer": "根据知识库中的 Redis 内存告警处置手册...",
  "citations": []
}
```

---

### 36.13 日志分析接口调整

原接口：

```http
POST /api/log-analysis
```

请求增加 conversationId：

```json
{
  "conversationId": 1001,
  "sourceId": 1,
  "question": "支付接口 9 点后超时增多，可能是什么原因？",
  "systemName": "支付系统",
  "componentName": "payment-api",
  "timeStart": "2026-07-05T09:00:00+08:00",
  "timeEnd": "2026-07-05T10:00:00+08:00",
  "keyword": "timeout OR ERROR",
  "logLevel": "ERROR",
  "topK": 5
}
```

处理规则：

```text
1. 必须登录。
2. 普通用户只能看到自己的日志分析任务。
3. admin 可以看到全部日志分析任务。
4. 日志分析结果写入 conversation_message。
5. 日志分析过程更新 task_state。
6. 工具调用记录写入 tool_call_record。
7. LLM 调用前保存 context_snapshot。
```

---

## 37. 上下文 Prompt 组装规则

### 37.1 Prompt 输入优先级

最终传给 LLM 的上下文按以下优先级组织：

```text
1. 系统安全规则
2. 当前用户问题
3. 当前会话任务状态
4. 当前工具结果摘要
5. 知识库检索结果
6. 最近多轮对话
7. 会话摘要
8. 用户基础信息
```

注意：

```text
当前用户问题优先于历史上下文。
当前工具结果优先于历史摘要。
已发布知识库内容优先于模型自身知识。
普通用户权限边界优先于一切业务逻辑。
```

---

### 37.2 知识库问答 Prompt 调整版

```text
你是一个资深银行生产运维专家。

你正在为一个已登录用户提供知识库问答服务。

【当前用户】
{{user_context}}

【当前会话摘要】
{{conversation_summary}}

【当前任务状态】
{{task_state}}

【最近对话】
{{recent_messages}}

【用户当前问题】
{{question}}

【知识库内容】
{{chunks}}

要求：
1. 请严格基于【知识库内容】和【当前任务状态】回答用户问题。
2. 不要编造知识库中不存在的信息。
3. 如果知识库没有相关依据，请明确说明：“知识库中未找到明确依据”。
4. 如果当前用户问题和历史上下文冲突，以当前用户问题为准。
5. 如果工具结果和历史摘要冲突，以工具结果为准。
6. 涉及生产命令时，只能作为排查建议展示，不允许描述为可以直接执行。
7. 涉及重启、删除、清理、扩容、切换、回滚等高风险操作时，必须提示需要按生产变更流程审批。
8. 回答要结构清晰。
9. 最后列出引用来源。
10. 不要输出用户的敏感信息、token、密码、私钥、连接串。

请按以下格式回答：

## 结论

## 依据

## 排查步骤

## 建议命令

## 风险提示

## 引用来源
```

---

### 37.3 日志分析 Prompt 调整版

```text
你是一个资深银行生产运维日志分析专家。

你正在为一个已登录用户分析日志和知识库内容。

【当前用户】
{{user_context}}

【当前会话摘要】
{{conversation_summary}}

【当前任务状态】
{{task_state}}

【最近对话】
{{recent_messages}}

【用户当前问题】
{{question}}

【日志来源】
{{log_source}}

【日志时间范围】
{{time_range}}

【日志样本】
{{log_samples}}

【知识库内容】
{{chunks}}

要求：
1. 必须区分日志中可以直接观察到的事实、基于知识库的依据、以及推测性的可能原因。
2. 不要编造日志中不存在的错误、时间点、接口、主机或指标。
3. 如果知识库没有相关依据，请明确说明：“知识库中未找到明确依据”。
4. 如果当前用户问题和历史上下文冲突，以当前用户问题为准。
5. 如果工具结果和历史摘要冲突，以工具结果为准。
6. 涉及生产命令时，只能作为排查建议展示，不允许描述为可以直接执行。
7. 涉及重启、删除、清理、扩容、切换、回滚等高风险操作时，必须提示需要按生产变更流程审批。
8. 输出需要包含日志证据和引用文档。
9. 不要输出用户的敏感信息、token、密码、私钥、连接串。

请按以下格式回答：

## 异常摘要

## 日志证据

## 可能原因

## 建议排查步骤

## 风险提示

## 知识库引用
```

---

## 38. 上下文维护流程

### 38.1 用户提问流程

```text
用户登录
    ↓
前端携带 JWT 请求 /api/qa/ask
    ↓
AuthMiddleware 解析当前用户
    ↓
校验 conversationId 是否属于当前用户
    ↓
保存用户问题到 conversation_message
    ↓
读取最近 6 到 10 轮对话
    ↓
读取 conversation_summary
    ↓
读取或创建 task_state
    ↓
执行知识库检索
    ↓
执行 LLM 重排
    ↓
组装 Prompt 上下文
    ↓
保存 context_snapshot
    ↓
调用默认 LLM
    ↓
保存 assistant 回答到 conversation_message
    ↓
保存 qa_record
    ↓
更新 conversation_summary
    ↓
更新 task_state
    ↓
返回答案和引用来源
```

---

### 38.2 会话摘要更新规则

触发条件：

```text
1. 当前 conversation_message 数量超过 10 条。
2. 最近消息总 token 估算超过 3000。
3. 每次日志分析或 K8s 诊断完成后。
4. 用户明确切换话题时。
```

摘要内容必须包含：

```text
1. 用户正在解决的问题
2. 已确认事实
3. 已使用的知识库文档
4. 已读取的日志或 K8s 上下文摘要
5. 已给出的结论
6. 待确认事项
7. 风险提示
8. 用户明确约束
```

摘要更新要求：

```text
1. 不要覆盖原始消息。
2. 摘要只作为上下文压缩，不作为唯一事实来源。
3. 摘要和当前工具结果冲突时，以当前工具结果为准。
4. 摘要和当前用户问题冲突时，以当前用户问题为准。
```

---

### 38.3 task_state 更新规则

每次问答后，调用 LLM 或规则逻辑更新 task_state。

task_state 更新 Prompt：

```text
你是一个运维排障任务状态维护助手。

请根据当前用户问题、助手回答、知识库引用、日志证据和已有 task_state，更新任务状态。

要求：
1. 只提取已经明确出现的信息。
2. 不要编造系统名、组件名、错误原因。
3. 如果当前问题改变了方向，需要更新 current_topic。
4. 保留已确认事实。
5. 标记待确认事项。
6. 输出 JSON，不要输出多余解释。

已有 task_state：
{{old_task_state}}

当前用户问题：
{{question}}

助手回答：
{{answer}}

引用来源：
{{citations}}

工具结果摘要：
{{tool_result_summary}}

输出格式：
{
  "taskType": "",
  "systemName": "",
  "componentName": "",
  "environment": "",
  "problemSummary": "",
  "knownFacts": [],
  "checkedItems": [],
  "currentConclusion": "",
  "pendingQuestions": [],
  "riskLevel": "low | medium | high | unknown",
  "nextActions": []
}
```

---

## 39. 后端目录结构扩展

在原有后端目录基础上增加：

```text
backend/
  internal/
    middleware/
      auth.go
      rbac.go

    model/
      user.go
      login_audit.go
      conversation.go
      conversation_message.go
      conversation_summary.go
      task_state.go
      tool_call_record.go
      context_snapshot.go

    handler/
      auth_handler.go
      user_handler.go
      conversation_handler.go

    service/
      auth_service.go
      user_service.go
      conversation_service.go
      context_service.go
      prompt_builder_service.go
      task_state_service.go

    repository/
      user_repository.go
      conversation_repository.go
      context_repository.go
      task_state_repository.go

    security/
      password.go
      jwt.go
```

职责说明：

```text
AuthService             登录、JWT 签发、修改密码
UserService             用户增删改查、启用禁用、重置密码
ConversationService     会话创建、查询、归档、消息保存
ContextService          最近消息、摘要、任务状态、工具结果聚合
PromptBuilderService    统一组装最终 LLM Prompt
TaskStateService        创建和更新 task_state
AuthMiddleware          校验 JWT
RBACMiddleware          校验 admin / user 权限
```

---

## 40. 前端目录结构扩展

在原有前端目录基础上增加：

```text
frontend/
  src/
    api/
      authApi.ts
      userApi.ts
      conversationApi.ts

    pages/
      LoginPage.tsx
      UserManagePage.tsx
      ConversationListPage.tsx

    components/
      auth/
        LoginForm.tsx
        RequireAuth.tsx
        RoleGuard.tsx

      user/
        UserTable.tsx
        UserFormDialog.tsx
        ResetPasswordDialog.tsx

      chat/
        ConversationSidebar.tsx

    stores/
      authStore.ts
```

前端路由要求：

```text
1. 未登录用户只能访问 /login。
2. 已登录用户访问 /login 时跳转到首页。
3. 普通页面需要 RequireAuth。
4. 用户管理页面需要 RoleGuard admin。
5. Header 显示当前用户名和角色。
6. 支持退出登录。
```

---

## 41. API 权限控制要求

所有接口默认需要登录，除了：

```text
GET /api/health
POST /api/auth/login
```

admin 专属接口：

```text
POST   /api/users
GET    /api/users
PUT    /api/users/{id}
POST   /api/users/{id}/reset-password

POST   /api/documents/{id}/review
POST   /api/llm-configs
PUT    /api/llm-configs/{id}
POST   /api/llm-configs/{id}/default

POST   /api/log-sources
PUT    /api/log-sources/{id}
DELETE /api/log-sources/{id}

POST   /api/k8s/clusters
PUT    /api/k8s/clusters/{id}
DELETE /api/k8s/clusters/{id}
```

普通用户可访问：

```text
GET    /api/documents
GET    /api/documents/{id}
POST   /api/documents/upload
POST   /api/qa/ask
GET    /api/conversations
POST   /api/conversations
GET    /api/conversations/{id}/messages
DELETE /api/conversations/{id}
POST   /api/logs/preview
POST   /api/log-analysis
POST   /api/k8s/diagnosis
```

数据归属规则：

```text
1. 普通用户只能查看自己的 conversation、qa_record、log_analysis_task、k8s_diagnosis_task。
2. admin 可以查看全部。
3. 普通用户不能修改其他用户的数据。
4. 后端必须基于 JWT 中的 userId 判断归属，不相信前端传入的 userId。
```

---

## 42. 安全要求补充

在原安全要求基础上增加：

```text
1. 所有业务接口默认必须登录。
2. 所有 admin 接口必须校验 role = admin。
3. 密码必须使用 bcrypt 哈希存储。
4. JWT_SECRET 必须从环境变量读取，不能硬编码。
5. 普通用户不能访问其他用户的会话上下文。
6. 普通用户不能查看其他用户的日志分析结果。
7. 普通用户不能查看其他用户的 K8s 诊断结果。
8. context_snapshot 中不得保存明文密码、token、私钥、连接串。
9. LLM Prompt 中不得注入用户密码、JWT、日志源凭据、K8s 凭据。
10. admin 禁用用户后，该用户不能继续登录。
11. 禁止删除最后一个 admin。
12. 禁止禁用最后一个 admin。
```

---

## 43. 第一阶段 MVP 范围调整

将原“可以暂缓”中的：

```text
用户登录
权限管理
```

从暂缓项移除，并加入第一阶段必须实现。

### 第一阶段必须实现

```text
1. React + shadcn/ui 前端基础页面
2. Golang + Gin 后端服务
3. PostgreSQL + pg_trgm 表结构
4. 简化用户登录
5. admin / user 两类角色
6. admin 管理普通用户
7. JWT 认证中间件
8. 基础 RBAC 权限控制
9. Markdown / TXT 文档上传
10. 文档切片
11. 检索增强信息入库
12. 知识库问答
13. 会话 conversation 管理
14. 对话消息 conversation_message 保存
15. 最近 6 到 10 轮上下文维护
16. conversation_summary 会话摘要
17. task_state 任务状态维护
18. context_snapshot 上下文快照
19. 可选 LLM 接口配置和调用
20. 引用来源展示
21. 文档质量检查
22. Elasticsearch 日志源配置
23. 服务器日志文件源配置
24. 日志预览
25. 基于日志和知识库的 LLM 分析
```

### 仍然可以暂缓

```text
1. PDF 解析
2. MinIO
3. 文档版本对比
4. Wiki 自动同步
5. 工单系统集成
6. 日志长期归档
7. 实时流式日志监控
8. 自动化处置闭环
9. 多租户复杂权限
10. 细粒度系统权限
11. 部门组织架构
12. LDAP / OAuth / SSO
```

---

## 44. Codex 开发任务拆分调整

请在原 Task 3 之后插入以下任务。

### Task 3.1：用户表和认证基础

目标：

```text
实现 app_user、login_audit 表，支持初始化 admin 用户。
```

验收标准：

```text
1. migrations 中包含 app_user、login_audit 表。
2. 后端启动时如果没有 admin，自动创建默认 admin。
3. 密码使用 bcrypt 保存。
4. 不允许保存明文密码。
5. JWT_SECRET 从环境变量读取。
```

---

### Task 3.2：登录接口和 JWT 中间件

目标：

```text
实现 /api/auth/login、/api/auth/me、/api/auth/change-password。
```

验收标准：

```text
1. 用户可以使用 username + password 登录。
2. 登录成功返回 accessToken 和用户信息。
3. 登录失败写入 login_audit。
4. 禁用用户不能登录。
5. 受保护接口必须携带 Authorization: Bearer token。
6. AuthMiddleware 可以解析当前用户。
```

---

### Task 3.3：admin 用户管理

目标：

```text
实现 admin 管理普通用户。
```

验收标准：

```text
1. admin 可以创建普通用户。
2. admin 可以查询用户列表。
3. admin 可以启用、禁用普通用户。
4. admin 可以重置普通用户密码。
5. 普通用户不能访问用户管理接口。
6. 系统不允许禁用或删除最后一个 admin。
```

---

### Task 3.4：会话和上下文表

目标：

```text
实现 conversation、conversation_message、conversation_summary、task_state、tool_call_record、context_snapshot 表。
```

验收标准：

```text
1. migrations 中包含上述表。
2. GORM model 与表结构对应。
3. conversation 绑定 user_id。
4. 普通用户只能访问自己的 conversation。
5. admin 可以访问全部 conversation。
```

---

### Task 3.5：ContextService 和 PromptBuilderService

目标：

```text
实现上下文聚合和 Prompt 组装。
```

验收标准：

```text
1. 可以根据 user_id + conversation_id 读取最近 6 到 10 轮消息。
2. 可以读取 conversation_summary。
3. 可以读取 task_state。
4. 可以合并知识库检索结果和工具结果摘要。
5. 可以生成最终 LLM Prompt。
6. LLM 调用前保存 context_snapshot。
7. 敏感信息不会进入 Prompt。
```

---

### Task 3.6：会话摘要和 task_state 更新

目标：

```text
实现会话摘要和任务状态维护。
```

验收标准：

```text
1. 问答完成后可以更新 conversation_summary。
2. 问答完成后可以更新 task_state。
3. 日志分析完成后可以更新 task_state。
4. 摘要不会覆盖原始消息。
5. 当前问题优先于历史摘要。
6. 工具结果优先于历史摘要。
```

---

### Task 10 调整：前端问答页面

原目标保持不变，增加：

```text
1. ChatPage 左侧展示当前用户的会话列表。
2. 用户可以新建会话。
3. 用户可以切换会话。
4. 用户提问时携带 conversationId。
5. 页面刷新后可以继续查看历史消息。
6. 普通用户只能看到自己的会话。
7. admin 可以查看全部会话或按用户筛选。
```

---

### 新增 Task 10.1：前端登录和用户管理页面

目标：

```text
实现 LoginPage、UserManagePage、RequireAuth、RoleGuard。
```

验收标准：

```text
1. 未登录用户访问业务页面会跳转到 /login。
2. 登录成功后保存 token。
3. Header 显示当前用户名和角色。
4. 支持退出登录。
5. admin 可以进入用户管理页面。
6. 普通用户看不到用户管理入口。
7. admin 可以新增、禁用、重置普通用户密码。
```

---

## 45. 最终验收标准补充

完成后，系统除原有能力外，还必须做到：

```text
1. 系统启动后自动初始化 admin 用户。
2. 用户必须登录后才能使用系统。
3. admin 可以创建和管理普通用户。
4. 普通用户不能访问用户管理页面。
5. 普通用户不能访问 admin 专属接口。
6. 普通用户只能看到自己的会话和分析记录。
7. admin 可以查看所有用户的会话和分析记录。
8. 用户可以创建多个聊天会话。
9. 每个会话可以保存多轮消息。
10. 用户切换会话后，上下文不会混淆。
11. 问答接口可以自动读取最近 6 到 10 轮消息。
12. 系统可以维护 conversation_summary。
13. 系统可以维护 task_state。
14. 每次调用 LLM 前保存 context_snapshot。
15. LLM Prompt 中包含当前用户、当前会话、最近消息、会话摘要、任务状态、知识库片段。
16. 日志分析结果可以写入当前会话上下文。
17. K8s 诊断结果可以写入当前会话上下文。
18. 所有操作记录当前 user_id。
19. 密码、token、私钥、连接串不会进入 LLM Prompt。
20. 用户 A 的上下文不会泄露给用户 B。
```
