# 运维知识库 RAG 系统

按 `features.md` 生成的第一阶段 MVP，包含：

- Golang + Gin + GORM 后端
- PostgreSQL + pg_trgm 表结构
- Markdown / TXT 文档上传、解析、切片、检索增强信息入库
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

## 注意

- 只有 `published` 状态文档会参与问答检索。
- 当前 MVP 优先支持 `.md` 和 `.txt`。
- LLM 生成的命令只作为排查建议展示，系统不会执行生产命令。
- 当前版本不需要 embedding 模型；检索链路使用 DeepSeek 查询改写、PostgreSQL `pg_trgm` 召回、DeepSeek 重排。
