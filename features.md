# 运维知识库 RAG 系统开发说明

## 1. 项目目标

开发一个面向运维场景的知识库问答系统。

系统基于内部运维文档、操作手册、告警处置手册、应急预案、变更回滚方案等资料，构建 RAG 知识库。

用户可以上传文档，并基于知识库提问。系统需要先检索知识库内容，再调用内网 DeepSeek v4 LLM 模型生成答案。

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

系统暂不实现：

1. 自动执行命令
2. 自动修改生产配置
3. 自动重启服务
4. 自动清理数据
5. 多租户复杂权限
6. 工单系统深度集成

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

    handler/
      document_handler.go
      qa_handler.go
      review_handler.go
      health_handler.go

    service/
      document_service.go
      parser_service.go
      chunk_service.go
      rag_service.go
      quality_service.go
      retrieval_metadata_service.go
      review_service.go

    repository/
      document_repository.go
      chunk_repository.go
      qa_repository.go

    client/
      deepseek_client.go

    dto/
      document_dto.go
      qa_dto.go
      review_dto.go

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

    pages/
      DashboardPage.tsx
      DocumentListPage.tsx
      DocumentUploadPage.tsx
      DocumentDetailPage.tsx
      ChatPage.tsx
      ReviewPage.tsx

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
```

说明：

如果内网 DeepSeek v4 接口兼容 OpenAI Chat Completions，则后端直接使用：

```text
POST /v1/chat/completions
```

如果不兼容，需要在 `deepseek_client.go` 中封装适配器。

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

## 13. LLM Prompt 设计

### 13.1 知识库问答 Prompt

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

### 13.2 文档质量检查 Prompt

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

## 14. DeepSeek v4 客户端要求

### 14.1 Chat 接口

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

### 14.2 LLM 用途

由于当前只有 DeepSeek v4 LLM，后端需要复用同一个 Chat 接口完成：

```text
1. 文档质量检查
2. chunk 检索增强信息生成
3. 用户问题查询改写和关键词抽取
4. 候选 chunk 重排
5. 最终 RAG 回答生成
```

---

## 15. 后端 Service 职责

### 15.1 DocumentService

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

### 15.2 ParserService

负责解析：

```text
.txt
.md
.pdf
.docx
.xlsx
```

第一版可以优先实现：

```text
.txt
.md
```

PDF、Word、Excel 可以预留接口，后续扩展。

---

### 15.3 ChunkService

负责：

```text
1. 按标题切片
2. 按段落切片
3. 控制 chunk_size
4. 控制 chunk_overlap
5. 生成 chunk_index
```

---

### 15.4 RAGService

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

### 15.5 RetrievalMetadataService

负责：

```text
1. 接收 chunk 内容
2. 调用 DeepSeek 生成摘要、关键词、可能问题
3. 生成 search_text
4. 写入 kb_chunk 的 search_text、keywords、possible_questions 字段
```

### 15.6 QualityService

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

## 16. 前端页面说明

### 16.1 DashboardPage

展示：

```text
1. 文档总数
2. 已发布文档数
3. 待审核文档数
4. 平均质量分
5. 最近问答记录
```

---

### 16.2 DocumentUploadPage

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

### 16.3 DocumentListPage

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

### 16.4 ChatPage

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

### 16.5 ReviewPage

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

## 17. shadcn/ui 组件建议

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

## 18. 前端 API 封装

### 18.1 documentApi.ts

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

### 18.2 qaApi.ts

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

## 19. 后端返回格式统一

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

## 20. 安全要求

1. 不允许系统自动执行 LLM 生成的命令。
2. 所有命令只能作为文本建议展示。
3. 涉及生产变更、重启、删除、清理、扩容、回滚时，必须提示审批。
4. 不允许把草稿、废弃、归档文档用于问答。
5. LLM 回答必须带引用来源。
6. 如果检索不到文档，必须明确说明知识库没有依据。
7. 文档上传大小需要限制，默认最大 50MB。
8. 文件类型需要白名单控制。
9. 后端需要记录问答日志，方便审计。

---

## 21. 第一阶段 MVP 范围

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
```

### 可以暂缓

```text
1. PDF 解析
2. Word 解析
3. Excel 解析
4. MinIO
5. 用户登录
6. 权限管理
7. 文档版本对比
8. Wiki 自动同步
9. 工单系统集成
```

---

## 22. Codex 开发任务拆分

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
1. 支持上传 .md 和 .txt
2. 文件保存到本地目录
3. 创建 kb_document 记录
4. 返回文档 ID
```

---

### Task 5：文档解析和切片

目标：

```text
解析 .md 和 .txt 文档，切分为 chunks。
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

## 23. 开发约束

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

---

## 24. 最终验收标准

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
```

---

## 25. 示例运维手册

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

## 26. 示例问题

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
