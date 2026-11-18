# ARTEX 使用手册

## 安装

> 依赖数据库 **PostgreSQL**；探索需配置 **LLM**（`ANTHROPIC_API_KEY` 或 `OPENAI_API_KEY`，也可在 UI 里配）。

### 方式一：一键安装脚本（推荐）

```bash
git clone https://github.com/RuoJi6/ARTEX.git
cd ARTEX
./install.sh
```

脚本会：检测 / 自动安装 Docker → 让你选 **① 全部 Docker** 或 **② 本地编译运行**：

- **① 全部 Docker**：填一个 Postgres 密码（可回车随机）→ 自动写 `.env` → `docker compose up -d`。
- **② 本地运行**：选数据库（连已有 / 用 Docker 起一个）→ 生成 `config.json` → `go` 编译内嵌单二进制 → 启动。

装好后打开 **http://localhost:8787**（首次进入 `/setup` 设置管理员密码）。

### 方式二：Docker Compose（手动）

```bash
git clone https://github.com/RuoJi6/ARTEX.git
cd ARTEX
cp .env.example .env          # 填 POSTGRES_PASSWORD、可选 ANTHROPIC_API_KEY
docker compose up -d          # 拉取 ruoji6/artex 镜像 + postgres
# → http://localhost:8787
```

镜像已含常用工具（ripgrep/curl/vim/npm/nmap…）；`./skills` 与 `./data` 以绑定挂载持久化。

`.env` 中默认 `ARTEX_IMAGE=ruoji6/artex`，`ARTEX_TAG=latest` 使用最新镜像，也可固定为 `v0.3.17`（完整镜像地址 `ruoji6/artex:v0.3.17`）。镜像支持 `linux/amd64` 和 `linux/arm64`，Docker 会自动选择与宿主机匹配的架构，ARM64 Kali 无需自行编译。初次部署请修改 `POSTGRES_PASSWORD`；运行服务无需配置 Docker Hub 发布 Token。

远程 MCP 可在系统设置中选择 `http`（Streamable HTTP）或 `sse`（旧版 SSE）。
旧版 SSE 服务通常使用 `GET /sse` 建立事件流，再通过服务返回的
`/message?sessionId=...` 接收 JSON-RPC 请求；配置时将 URL 填为 `/sse`，请求头按
`Authorization=Bearer <token>` 填写。

### 方式三：下载预编译二进制（Releases）

到本仓库的 [Releases](https://github.com/RuoJi6/ARTEX/releases) 下载对应平台的 zip，解压后得到 `artex` + `start.sh`（Windows 为 `start.bat`）+ `skills/` + `config.example.json`。Linux ARM64（包括 ARM64 Kali）选择 `linux-arm64`，x86_64 选择 `linux-amd64`：

```bash
cp config.example.json config.json   # 填好 database 连接
./start.sh                           # → http://localhost:8787
```

> 请用 `start.sh` / `start.bat` 启动，而不是直接跑 `./artex`。它是个守护脚本：程序退出后按退出码决定是否重新拉起，**页面上的[一键更新](#方式一页面一键更新推荐)靠它完成换装**。直接运行 `./artex` 时更新完就不会被拉起了。
> 后台常驻：`nohup ./start.sh >artex.log 2>&1 &`。

### 方式四：从源码编译单二进制

```bash
# 1) 前端静态导出
cd web && npm ci && npm run build:static && cd ..
# 2) 创建内嵌目录
mkdir -p server/webui/dist
# 3) 拷进内嵌目录
cp -r web/out/. server/webui/dist
# 4) 编译（-tags embedui 才内嵌前端）
CGO_ENABLED=0 go build -tags embedui -o artex ./cmd/artex
./start.sh
```

---

## 更新升级

> 升级只换程序、不动数据：Postgres 数据卷 `pgdata`、`./data`（jwt.key / SQLite 等）、`./skills` 都会保留。**数据库迁移无需手动执行**——`artex` 每次启动会幂等重跑 `schema.sql`（含 `ADD COLUMN` / `CREATE INDEX IF NOT EXISTS`），即“重启即迁移”。升级前仍建议先备份 `./data` 与数据库。

### 方式一：页面一键更新（推荐）

在 **系统配置** 页（侧边栏「系统配置」→ `/system/settings`）的**版本与更新**卡片里，可以直接检查并安装新版本，无需登录服务器。

本仓库构建的程序固定从 `RuoJi6/ARTEX` 的最新正式 GitHub Release 检查更新，跳过草稿和预发布版本。旧版官方程序仍指向原作者仓库，因此首次切换必须手动安装本仓库构建的程序；以后发布更高的三段式版本即可通过页面升级。仅推送 main 或 Git 标签但未发布 Release，不会产生可在线安装的新版本。

点「更新」后：下载当前平台的发布包 → 比对 Release 的 `SHA256SUMS` → 用 `-h` 冒烟测试新二进制 → 暂存为 `artex.new` → 程序退出，由 `start.sh` / `start.bat` 重新拉起并完成换装。页面会自动等到新版本上线后刷新。

- **失败不会留下坏程序**：校验或冒烟不通过就丢弃暂存件、继续跑当前版本；换装后的新版若连续 3 次启动失败，会自动回滚到 `artex.old`（失败的那个留作 `artex.failed` 供排查）。
- **随时可回退**：上一版本保留为 `artex.old`，卡片上有「回滚到上一版本」。注意数据库结构不会回退。
- **更新会中断正在运行的任务**——更新即重启，请在空闲时进行。
- **开发构建不给更新**：版本号是 `dev` 或 `git describe` 带后缀时禁用，避免正式版覆盖掉本地调试的二进制。
- **Docker 下只换程序、不换镜像**：镜像里的 playwright / nmap 等工具链不会跟着升级，且 `docker compose up -d` 重建容器后会退回镜像自带的版本。要连镜像一起升级仍请用 `docker compose pull artex && docker compose up -d artex`。
- **更新代理（可选）**与其他代理设置使用相同的「地址输入框 + 保存」方式。填写独立更新代理后保存，例如 `http://127.0.0.1:7897`；支持 `http://`、`https://`、`socks5://`、`socks5h://` 及用户名密码认证。清空并保存即为直连，不受全局代理或 `HTTP_PROXY` / `HTTPS_PROXY` 环境变量影响。保存后自动重新检查，检查版本、校验和下载和发布包下载使用同一个出口。更新只从 GitHub 域名下载并强制 HTTPS。
- 代理地址由 **ARTEX 服务端**访问。Docker 内的 `127.0.0.1` 指容器自身；代理运行在宿主机时，应使用容器可达地址（Docker Desktop 通常为 `http://host.docker.internal:7897`），并允许容器访问代理端口。

### 已有 Docker 部署迁移到本仓库镜像

在原部署目录中备份 `.env`、Compose 文件、`data/` 和 PostgreSQL 数据，保留原数据库卷和 skills 挂载。将 `.env` 的 `ARTEX_IMAGE` 设置为本仓库镜像地址、`ARTEX_TAG` 设置为目标版本；如果旧 Compose 仍写死原作者的镜像，可创建下列覆盖文件（已有覆盖文件时应合并 image 配置）：

```yaml
# docker-compose.override.yml
services:
  artex:
    image: ${ARTEX_IMAGE}:${ARTEX_TAG:-latest}
```

然后只更新应用容器：

```bash
docker compose pull artex
docker compose up -d --no-deps artex
curl --fail http://127.0.0.1:8787/api/health
docker compose logs --tail=50 artex
```

继续使用原部署目录和 Compose 项目名称，确保 `./data`、`./skills` 和 PostgreSQL 数据卷仍指向原数据。迁移后程序从 `RuoJi6/ARTEX` 检查在线更新；后续可用页面更新程序或上述命令更新整个镜像。

首次迁移已有 Docker 部署时，将已校验的新 ARM64 ZIP 解压到宿主机，备份原程序，然后只替换已有容器中的 `/app/artex`。例如现有容器名为 `artex-artex-1`：

```bash
# 在下载目录运行；先按 SHA256SUMS 核对下载的 ZIP，再解压。
sha256sum --ignore-missing --check SHA256SUMS
unzip artex-0.3.17-linux-arm64.zip
sudo docker stop artex-artex-1
sudo docker cp artex-artex-1:/app/artex ./artex-before-migration
sudo docker cp ./artex-0.3.17-linux-arm64/artex artex-artex-1:/app/artex
sudo docker start artex-artex-1
curl --fail http://127.0.0.1:8787/api/health
```

这会保留现有数据库、data 和 skills 挂载。登录系统配置页检查更新源与版本。容器内替换的程序仍属于容器可写层，删除或重建容器会丢失替换结果；需要重建时应使用包含本仓库程序的镜像。程序回滚不回退数据库迁移，升级前请备份数据库和 data。

### 方式二：一键更新脚本

```bash
cd ARTEX
./update.sh
```

脚本先可选 `git pull` 拉取最新代码，再让你选 **① Docker 更新** 或 **② 本地编译更新**（与 `install.sh` 对应）：

- **① Docker**：可指定目标镜像 tag（回车沿用 `.env` 的 `ARTEX_TAG`，缺省 `latest`）→ `docker compose pull` → `docker compose up -d`（换新镜像重启即自动迁移）。
- **② 本地**：重建前端静态产物 → 重新编译 `./artex`（完成后重启进程生效）。

### 方式三：Docker Compose（手动）

```bash
cd ARTEX
git pull                       # 更新 compose / 脚本（可选）
# 指定版本：在 .env 设 ARTEX_TAG=v0.3.17；不设则用 latest
docker compose pull artex
docker compose up -d artex     # 换新镜像重启 → 自动迁移 schema
docker image prune -f          # 清理旧镜像（可选）
```

### 方式四：预编译二进制（Releases）

到 [Releases](https://github.com/RuoJi6/ARTEX/releases) 下载新版本 zip，停掉旧进程后覆盖 `artex` 与 `skills/`（保留你的 `config.json` 与 `data/`），重启即可：

```bash
cp -r <解压目录>/skills ./ && cp <解压目录>/artex ./
./start.sh
```

### 方式五：从源码编译

```bash
git pull
cd web && npm ci && npm run build:static && cd ..
mkdir -p server/webui/dist
cp -r web/out/. server/webui/dist
CGO_ENABLED=0 go build -tags embedui -o artex ./cmd/artex
# 重启 ./start.sh
```

---

## 配置

**数据库**（`config.json`，或用环境变量 `ARTEX_PG_DSN` 覆盖）：

```json
{
  "database": {
    "host": "127.0.0.1", "port": 5432,
    "user": "artex", "password": "yourpass",
    "dbname": "artex", "sslmode": "disable"
  }
}
```

**LLM**：`export ANTHROPIC_API_KEY=sk-...`（或 `OPENAI_API_KEY`），也可在 UI 的「LLM 配置」页填写。
可选：`ARTEX_LLM_PROVIDER` / `ARTEX_LLM_MODEL` / `ARTEX_LLM_BASE_URL` / `ARTEX_LLM_PROXY`。

**HTTP User-Agent**：内置 WebFetch（worker / planner / main / chat）、资产 HTTP 自动补全、出口连通性探测、HTTP 自定义工具、请求重放器、GitHub 自更新和可选的 `harvest_static.py` / `spider_mpa.py` 默认从 20 个浏览器 UA 样本中随机抽选，不携带工具名称。Go 每次逻辑请求选择一次，重定向及 WebFetch 的代理失败直连重试复用同一个值；Python 脚本每次运行选择一次，入口、下载与重试保持一致。随机抽选允许重复，不依赖 AI 服务。通过 `ARTEX_HTTP_USER_AGENT` 可指定固定值；留空或非法值使用随机策略。直接运行时导出环境变量并重启服务，Docker 在 `.env` 配置后重建容器。Python 的 `--user-agent VALUE`、自定义工具和重放器中的显式 UA 优先，显式空值仍可省略此头。值去除首尾空格，仅接受可打印 ASCII。记录代理保留已有 UA 及显式省略语义。UA 随机化不代表 TLS、Cookie 或其他浏览器特征同步改变，也不保证无法识别客户端。此设置不影响浏览器、SDK 联网搜索、LLM / MCP / Webhook 服务通信或外部独立工具；本仓库未包含 phpfinger，其 `--user-agent` 需在该工具中单独配置。

完整排查范围、额外发现及未接管的服务通信见 [HTTP 请求标识排查](HTTP_USER_AGENT.md)。

**并发**：每个任务的 work agent 数在「系统设置」里配置（默认 3）。

**常用参数**：`./start.sh -addr :8787 -proxy :8788`（`-addr` 前端+API，`-proxy` 流量录制代理）。启动脚本会把参数原样透传给 `artex`。

### 可选的 HTTP Basic Auth 访问验证

在 **系统配置 → HTTP Basic Auth** 中填写独立的验证用户名和密码，打开「启用访问验证」并点击「保存访问验证」。默认关闭，保存后立即生效，无需重启；密码至少 8 位、最多 72 字节，用户名不能包含冒号。

开启后，访客先完成浏览器原生账号密码验证，再进入原有 ARTEX 登录页。网页、静态资源和 API 均受保护（MCP 服务端的 `/mcp` 端点除外：它不经过 Basic Auth，也不能用 Basic 账号密码访问，只认访问令牌，见下文「MCP 服务端」），`/api/health` 的 GET/HEAD 健康检查同样需要通过 Basic Auth，未认证时返回 `401`；通过后仅返回 `{"ok":true}`，不暴露应用名称或版本。健康检查不额外要求管理员 JWT；外部探针需要配置相同的 Basic Auth 凭据，例如使用 `curl --fail --user '验证用户名' http://127.0.0.1:8787/api/health` 并按提示输入密码。未启用 Basic Auth 时保留原有的无需凭据探活行为。版本号由需要管理员 JWT 的 `/api/version` 提供，启用访问验证后该接口同时受 Basic Auth 保护。未认证请求的 Basic Auth realm 使用通用名称 `Restricted`；通过访问验证仍需登录 ARTEX 才能使用业务 API。公网部署请通过 HTTPS 访问。

从仍使用 `/api/health` 读取版本的旧版网页升级到此版本时，旧页面可能无法自动判断升级完成；服务重启后手动刷新页面即可。新版网页的版本显示和升级轮询均使用 `/api/version`。

配置持久化到数据库，密码仅保存 bcrypt 哈希，界面不会回显。保存时密码留空表示不修改；关闭后保留凭据，方便再次开启。独立的 HttpOnly 验证 Cookie 有效期为 12 小时，兼容现有 JWT 和 SSE。验证过期时，后台请求会触发当前页面重新验证，保留 ARTEX 登录状态，无需新开标签页；保存配置会使其他浏览器的旧验证 Cookie 失效，当前管理员浏览器保持访问。浏览器可能缓存 Basic Auth 凭据，可用新的无痕窗口检查验证弹窗。

Docker 和内嵌前端二进制直接支持此功能。`next dev` 的认证检查仅向本机回环后端转发凭据（`localhost`、`127.0.0.1` 或 `::1`）；前端与后端应使用同一主机名。远程开发后端未开启验证时仍可使用；开启后请通过内嵌前端访问远程服务，避免跨域验证 Cookie 问题。

若遗忘独立验证密码，可在服务器连接应用数据库后执行以下语句并重启 ARTEX，关闭这层访问验证，再用原有 ARTEX 账号登录后重新配置：

```sql
UPDATE settings
SET value = jsonb_set(value::jsonb, '{enabled}', 'false'::jsonb)::text,
    updated_at = now()
WHERE key = 'auth.http_basic';
```

### MCP 服务端（供外部智能体编排任务）

ARTEX 可以作为 MCP 服务端，让外部智能体（个人助手、bot 等）查看任务与漏洞、创建任务、暂停或恢复任务。

注意区分：「系统 → MCP」页面管理的是 ARTEX 自己的 agent **去调用**的外部 MCP；本文说的是 ARTEX **对外提供**的 MCP 端点。

#### 开启与接入

1. 「系统设置 → 系统与账号 → MCP 服务端」打开开关。默认关闭，关闭时 `/mcp` 返回 404。
2. 在同一处「新建令牌」，选择权限范围。令牌只显示一次。
3. 在智能体里配置：

| 项 | 值 |
| --- | --- |
| 地址 | `https://<你的 ARTEX 地址>/mcp` |
| 传输 | Streamable HTTP（无状态，只响应 `application/json`，不提供 SSE，不支持 stdio） |
| 鉴权 | 请求头 `Authorization: Bearer artex_mcp_...` |

```bash
curl -s https://artex.example.com/mcp \
  -H 'Authorization: Bearer artex_mcp_xxx' -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

#### 无法使用 HTTP Basic Auth

`/mcp` **不经过** HTTP Basic Auth 验证，也**不接受** Basic 账号密码、ARTEX 登录密码或管理员登录后的 JWT，只认访问令牌。

原因：一个 HTTP 请求只有一个 `Authorization` 头，外部智能体无法同时带上 Basic 凭证和 Bearer 令牌。因此：

- 即使启用了 HTTP Basic Auth，`/mcp` 仍可直接凭访问令牌访问，Basic Auth 不会给它加一层保护。
- 需要限制谁能连 `/mcp` 的话，请用下文的「来源白名单」，不要依赖 Basic Auth。
- 公网暴露请使用 HTTPS，令牌是明文放在请求头里的。

#### 来源白名单

在「MCP 服务端」卡片的「来源白名单」里填写允许调用 `/mcp` 的来源地址，每行一个 IP 或 CIDR（如 `203.0.113.7`、`10.0.0.0/8`，支持 IPv6）。留空表示不限制来源。

- 白名单在令牌之前检查：不在名单内的来源即使持有有效令牌也会被拒绝，返回 `403`，响应里带有服务器看到的来源地址，便于排查。
- 白名单只作用于 `/mcp`，不影响网页和管理 API，填错不会把自己锁在后台之外。
- **部署在反向代理之后**：此时直连地址永远是代理。需要把代理地址填进「可信代理」，ARTEX 才会改用 `X-Forwarded-For` 里的地址作为来源（从右往左取第一个不属于可信代理的地址）。不填可信代理时该请求头一律忽略，避免调用方伪造来源。代理需要把真实来源追加到该请求头，例如 nginx 的 `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`。
- 卡片里会显示「服务器看到的你当前的来源地址」，可用来确认可信代理是否配置正确。

#### 权限范围与工具

令牌只能看到、调用它的权限范围内的工具。

| 权限范围 | 工具 | 说明 |
| --- | --- | --- |
| `read` | `list_tasks` | 列出任务，可按 `company_id`、`category_id`、`status` 筛选 |
| `read` | `get_task` | 单个任务的状态、运行时长、目标完成数、漏洞数；用它轮询进度 |
| `read` | `get_task_graph` | 任务的探索图总览 |
| `read` | `list_task_findings` | 任务的确认漏洞（不含原始请求/响应流量） |
| `read` | `list_companies` / `list_task_categories` / `list_llm_profiles` | 建任务时可选的企业、分类、模型配置（不含 API Key） |
| `task:create` | `spawn_task` | 新建并启动任务，可带 `company_ids`、`category_id` |
| `task:control` | `pause_task` / `resume_task` / `add_task_hint` | 暂停、恢复、补充提示 |

ARTEX 没有独立的「项目」实体。智能体说的“某个项目”对应某个企业或某个任务分类下的一组任务。

不会暴露的能力：建/改自定义工具、MCP、skill，workspace 文件读写，LLM 密钥，系统设置。

#### 令牌的限制项

- **限定企业**：勾选后，令牌只能在这些企业下建任务（必须带 `company_ids`，且目标地址须归属这些企业的范围），也只能看到、控制这些企业下的任务，其余任务对它表现为不存在。这是把任务绑定到企业的资产范围，不是网络层拦截；要硬性限制任务能访问什么，请配合资产拦截规则。
- **未结束任务上限**：该令牌创建的、尚未结束的任务数（含暂停和排队）。
- **每小时建任务上限**：按该令牌过去一小时创建的任务数计，任务删除后仍计入。
- **有效期**与**吊销**：吊销立即生效，不可恢复。

#### 审计

每次工具调用都会记录：令牌、工具、参数、是否成功、失败原因、来源地址、时间，包括因权限不足被拒绝的调用。在设置页的「调用审计」查看，保留最近 20000 条。

#### 安全提示

- 创建任务会对目标发起真实的安全测试，只把 `task:create` 令牌交给可信的智能体，并尽量限定企业、设置限额。
- 任务描述、漏洞摘要等返回内容可能包含来自被测目标的文本，对智能体而言是不可信输入，可能夹带提示注入。
- 带 `Origin` 头的跨源浏览器请求会被拒绝（403），`/mcp` 不发送 CORS 头。

### 反向代理部署（HTTPS / 只开放 443）

前端和 API/SSE 都由同一个后端端口（默认 `:8787`）提供，实时活动流默认走**同源**地址，因此**无需配置 `NEXT_PUBLIC_SSE_BASE`**，公网只开放 443、把 8787 留在内网即可。

SSE 是长连接 + 持续推送，反代**必须关闭缓冲**，否则浏览器能连上却收不到事件（表现为活动流一直转圈）。Nginx 示例：

```nginx
server {
    listen 443 ssl;
    server_name your.domain.com;
    # ssl_certificate / ssl_certificate_key ...

    location / {
        proxy_pass http://127.0.0.1:8787;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE 关键项：关缓冲、长超时、HTTP/1.1
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

> 仅当 SSE 需要走与页面不同的来源（如独立子域）时，才在**构建期**设置 `NEXT_PUBLIC_SSE_BASE`（该变量在 `next build` 时固化进静态包，容器运行时再设无效）。

---

## 日常使用

### 查看审批记录

在全局「审批记录」、任务内「拦截审批」或对话中的审批卡片上，展开记录即可查看详情。

### 资产同步（ScopeSentry）

支持从 [ScopeSentry](https://github.com/Autumn-27/ScopeSentry) 直接同步资产数据，免去重复收集：

- 在「**资产同步**」页填 ScopeSentry 的地址与 API Key，接入数据源；
- 按**项目**或**任务**维度选择要同步的目标与资产类型（域名 / 子域 / IP / 端口 / 站点 / 端点…）；
- 一键导入并按公司资产范围归并，直接进入 ARTEX 的资产图供 agent 探索使用。

### 同一漏洞归并与统一报告

发现页默认「按漏洞」。原始上报、按任务及按资产保留原来的表格、状态和操作按钮，仅已归并记录改为可展开的漏洞文件夹。同一任务内，Reporter 依据目标、接口参数、权限边界、根因及验证证据确认同一实际缺陷后形成文件夹；仅类型或名称相同不会作为自动归并依据。严重程度、接口、后端或读写影响不同仍可能是同一漏洞；反之，共用登录入口或令牌不能单独证明同一根因。有关联但共同根因无法确认时提交人工建议。文件夹的 `严/高/中/低` 数量按原始记录统计，例如 `0/1/1/1`；发现页的独立漏洞数按文件夹去重，等级数量按原始上报统计，列表及文件夹详情不显示统一评级标签。原始上报内容、评级、状态、报告及证据不被覆盖。

点击箭头展开原始记录，点击文件夹标题查看统一报告、原始记录及归并历史。证据不足的建议需要点击「确认归并」；否决或移出后，这些关系不会被 Agent 自动再次归并。移出成员不删除原漏洞，不足两条时解散文件夹并保留历史。继承任务的漏洞组保持只读。

新上报由现有 Reporter 生成独立报告并检查重复。升级首次补充 Reporter 的归并工具和运行指引，保留自定义提示词及触发设置；已删除或停用的 Reporter 不会重建、启用。历史记录不会自动扫描：展开并选择原始记录，点击「整理所选」，按任务分别排队执行，可查看执行记录及失败信息。同一任务已有排队或运行中的整理时拒绝重复提交。历史整理以只读方式复用原报告，禁止覆盖原报告或补绑证据；已有报告为空或过期时如实说明，单独通过报告撰写流程处理。结束前需提交每条所选记录的根因判断，检查读取范围、建议、统一报告完整性及版本，通过验收才显示完成。文件夹详情可重新生成统一报告；需要启用 Reporter 并配置可用模型。判断和写作会消耗模型 token，候选检索先分页返回摘要，详情证据按需读取，不重新探测目标。

成员、报告或证据变化后统一报告标记待更新，写入时校验实际读取版本。导出默认每组一份统一报告及未归并记录，可勾选「包含原始子报告」；统一报告未生成或过期时附明确说明及全部原始材料。原始上报显示所属任务、资产及时间；按任务先展开任务，再展开漏洞文件夹；按资产保留资产树，命中任一成员即展示整个文件夹，展开时标明未命中筛选的成员。折叠状态只读取轻量摘要，不加载成员报告。升级沿用启动时幂等数据库迁移，无需执行独立 SQL。

### 手动漏洞复测

任务详情的「复测」页签可分页选择本任务的漏洞、查看历次结论和证据，并手动发起复测。启动后保留当前页签，显示转圈图标和「复测中」；确认修复后同步更新漏洞状态。

在漏洞列表每行操作区点击「复测」，或在漏洞详情的「漏洞复测」区域点击「发起复测」，填写可选的修复版本、测试条件或限制，系统会创建独立的复测 Agent 会话，启动后保留当前页面。列表的平铺、按任务分组和资产视图均支持该入口；复测运行时显示转圈图标和「复测中」，需要查看时点击进入对应会话，结束后恢复「复测」。复测无需重新启动原扫描任务，结论分为「仍可复现」「已修复」「无法确认」，每次的结论、证据和会话链接保存在漏洞详情中。

新版后端首次启动会预置可编辑的「漏洞复测」（`retester`）Agent，可在 Agent 管理中配置提示词、LLM、运行预算和工具。默认使用其绑定的 LLM，未绑定则使用全局激活配置。复测会话成功完成且结论为「已修复」时，系统自动将漏洞处置状态改为「已修复」；执行中、失败、停止或其他结论保留原状态。原始证据和报告始终保留。也可在状态下拉菜单中手动选择「已修复」。同一漏洞正在复测时复用已有会话，停止、失败或服务重启后可重新发起。

本版历史记录通过漏洞详情和会话查看，暂未纳入漏洞报告导出或任务归档包，也未自动关联流量包。演示模式只生成明确标注的模拟记录，不请求真实目标。

### 自定义 IP 探测服务

在「系统配置 → 流量与安全」展开「自定义 IP 探测服务（可选）」，填写可访问的 HTTP / HTTPS 回显服务后点击「保存探测服务」。服务需返回纯文本 IP 或 `{"ip":"1.2.3.4"}` 格式的 JSON。支持内网地址，保留对本机、链路本地地址及跨主机重定向的限制。

保存后，顶栏出口 IP、代理测试、代理池健康检查和自动重连均使用该地址，刷新页面或重启后仍生效；自定义服务失败时不会再尝试默认公网服务。清空地址并保存可恢复默认的 api.ipify.org / ifconfig.me。修改后也可先点击「测试代理」临时验证，测试不会保存地址。

### 任务情报

在「系统配置 → 任务情报」开启 **Agent 自动记录**，即可在任务详情的「任务情报」中集中查看凭据、重要线索和敏感数据。此开关默认关闭；关闭时不向 Agent 注入情报工具、附加字段或提示，不启动额外总结 Agent，已有记录仍可查看。开启会增加提示及上报的 Token 消耗，对后续 Agent 执行生效；已在运行的会话可能仍保留旧工具描述，关闭后服务端立即拒绝新增情报。

页面支持分类、标题/上报者搜索、状态筛选、分页、状态更新及来源查看。凭据、敏感数据的正文和来源证据直接展示在列表及详情中，无需额外点击显示。直接关联任务的记录标明继承且只读；在资产页选择一项资产后点击「关联情报」可查看同一份记录。情报随任务归档、恢复和删除；新版归档格式为 v4，仍支持读取 v1–v3。此功能不自动补提取历史会话。
