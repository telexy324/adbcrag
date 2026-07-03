# 运维知识库 RAG 系统

按 `features.md` 生成的第一阶段 MVP，包含：

- Golang + Gin + GORM 后端
- PostgreSQL + pg_trgm 表结构
- Markdown / TXT / Word / Excel 文档上传、解析、切片、检索增强信息入库
- DeepSeek v4 兼容 OpenAI Chat Completions 调用
- DeepSeek 查询改写、候选片段重排和最终回答
- 知识库问答与引用来源展示
- 文档质量检查与审核发布流程
- React + Vite + TypeScript 前端

## 后端启动

```bash
cd backend
cp .env.example .env
go mod tidy
go run cmd/server/main.go
```

数据库需提前创建，并启用 `pg_trgm`。也可以执行：

```bash
psql "$DATABASE_URL" -f migrations/001_init.sql
```

如果已经执行过旧版 pgvector 迁移，请再执行：

```bash
psql "$DATABASE_URL" -f migrations/002_llm_only_retrieval.sql
```

健康检查：

```bash
curl http://127.0.0.1:8080/api/health
```

## 前端启动

```bash
cd frontend
pnpm install
pnpm dev
```

默认访问：

```text
http://127.0.0.1:5173
```

## Docker Compose 启动

```bash
export DEEPSEEK_BASE_URL=http://deepseek-v4.internal.local/v1
export DEEPSEEK_API_KEY=local-key
export DEEPSEEK_MODEL=deepseek-v4
export VITE_UPLOAD_TIMEOUT_MS=600000
export API_PROXY_READ_TIMEOUT=600s
export CORS_ALLOW_ORIGINS='*'
docker compose up --build
```

默认访问：

```text
http://127.0.0.1:5173
```

Compose 会启动：

- `frontend`：nginx 托管前端，并将 `/api` 反代到后端
- `backend`：Golang API 服务
- `postgres`：PostgreSQL 16，数据持久化到 `postgres_data`

上传接口耗时较长时，可以调整：

```bash
VITE_UPLOAD_TIMEOUT_MS=900000 API_PROXY_READ_TIMEOUT=900s docker compose up --build
```

## 注意

- 只有 `published` 状态文档会参与问答检索。
- 当前 MVP 支持上传 `.md`、`.txt`、`.doc`、`.docx`、`.xls`、`.xlsx`；`.docx/.xlsx/.xls` 通过 Go 库解析文本后送入 LLM 处理链路，`.doc` 会提示先转换为 `.docx`。
- LLM 生成的命令只作为排查建议展示，系统不会执行生产命令。
- 当前版本不需要 embedding 模型；检索链路使用 DeepSeek 查询改写、PostgreSQL `pg_trgm` 召回、DeepSeek 重排。
