# XNode 产品与迭代记录

更新日期：2026-11-11。本文是项目交接入口，事实和测试结果以对应提交为准。

## 目标与范围

XNode 对接官方 Xboard 面板，保留面板推送和轮询的自动更新能力。落地机可在本地 `config.yml` 配置 Trojan TCP+TLS fallback；面板更新用户、节点端口、路由等数据时，本地 fallback 规则继续进入生成的内核配置。兼容性检查应拒绝把配置有 fallback 的节点改成非 Trojan、非 TCP 或 Reality，避免误以为 fallback 仍有效。

同时持续核对 Xray-core、sing-box 官方代码及 Xboard 面板接口，逐项升级、测试，不以版本号较新作为合并依据。

## 代码基线

- 本 fork：`PiPi-happy/XNode`，`dev` 分支，当前基线 `7abd6d5`，发布标签 `v1.14.1`。
- 上游节点后端：`cedar2025/Xboard-Node`，`dev` 分支，审查时 `618263a`。本 fork 已增加机器模式、证书、热更新等大量代码，不能直接用上游覆盖。
- 官方面板：`cedar2025/Xboard`，核对版本 `d826a3b`。
- 当前 `go.mod` 使用 Go `1.27`、Xray-core `v1.260327.1-0.20261031132019-a4a3dff5b20c` 和 sing-box `v1.14.4`；两个内核均直接使用官方模块，不再依赖 fork 替换。

## 迭代 1：本地 fallback

### 配置契约

```yaml
kernel:
  type: xray # 也支持 singbox
  fallbacks:
    - dest: "127.0.0.1:8080"
    - alpn: h2
      dest: "127.0.0.1:8082"
```

`dest` 必须是明确的 `host:port`；空 `alpn` 是默认目标。`http/1.1` 使用默认目标，`h2` 使用指定目标。每个 ALPN 只允许一条规则。Xray 使用 inbound `settings.fallbacks`；sing-box 使用 `fallback` 和 `fallback_for_alpn`。有 ALPN 规则时，TLS 会通告相应 ALPN；默认目标与 h2 共存时也通告 `http/1.1`。回落目标必须由独立的 HTTP 服务提供，XNode 不负责安装网站。sing-box 会为配置的回落目标生成优先直连规则；若目标是本机管理端口，已认证的代理用户也可能访问该端口，因此只应指向可公开访问的回落网站端口。

配置属于本地 `kernel`，不从 Xboard API 读取。顶层设置由多实例继承，静态多节点可通过 `nodes[].kernel.fallbacks` 覆盖。机器模式目前共用实例的规则；若同一机器上不同节点需要不同回落目标，应使用多个实例或后续增加按节点 ID 的映射。

### 验收标准

- 解析无效地址、端口或重复 ALPN 时拒绝启动。
- 面板推送或轮询更新兼容的 Trojan 节点后，生成的两种内核配置仍包含本地 fallback。
- 面板改为不兼容协议、传输或 Reality 时拒绝该更新，并保留已运行配置。
- HTTP/1.1 与 h2 的 HTTPS 请求分别到达本地目标；Trojan 正确用户仍能连接。

### 当前状态

- [x] 本地配置、双内核生成、兼容性校验与单元测试；两种实际内核的配置解析器均接受生成结果。
- [x] 本机真实连接集成测试：两种内核的 HTTP/1.1 与 h2 请求均到达预期回落服务，正确密码的 Trojan 连接也成功。测试命令：`go test -tags integration -timeout 30s ./internal/kernel/integration`。
- [x] 香港 Linux 实机从公网请求 `gcphkb.dodoasclothing.uk:10221`：TLS SNI 下 HTTP/1.1 与 h2 均回落到主站并返回 200；从 Xboard 当前用户集读取凭据，标准 Trojan 认证和 TCP 转发成功（目标站返回 301）。Nginx 回源分别监听 `127.0.0.1:8080` 和 `127.0.0.1:8082`。
- [x] Xboard 将节点端口从 `10121` 改为 `10221` 后，XNode 已拉取新端口；本地 fallback 规则保持不变，健康检查和新端口下 HTTP/1.1、h2、有效 Trojan 请求均通过。错误 Trojan 密码测试返回 400。
- [x] 用户已通过浏览器访问 `gcphkb.dodoasclothing.uk:10221` 并确认显示主站。不同客户端对错误 Trojan 凭据的探测行为仍不作为 HTTP fallback 验收条件。

## 迭代 2：内核与面板适配

### 已核对

- 官方 Xboard `ServerService::buildNodeConfig` 仍发送 `protocol`、`networkSettings`、`tls_settings`、`custom_outbounds`、`custom_routes`、`cert_config` 等字段；当前 `internal/panel/types.go` 可接收这些字段。
- Xboard 扩展配置的代表性 JSON 契约测试覆盖网络设置、TLS、路由、自定义出站、证书和 multiplex/brutal；AnyTLS `padding_scheme` 的字符串和数组格式均有测试。真实运行面板 API 响应的脱敏样本尚未归档。协议列表审查发现 Xray 已有 HTTP 与 SOCKS inbound 生成器但运行时支持列表遗漏，已补齐列表和回归测试。
- 官方 Xboard 的机器节点接口使用 `POST /api/v2/server/machine/nodes`；fork 的机器模式代码已改为 POST。
- 官方 sing-box Trojan inbound 定义 `fallback` 和 `fallback_for_alpn`；官方 Xray-core Trojan/VLESS 定义 `fallbacks`。
- 升级前直接切换官方 sing-box 时会因 fork 热更新 API 不兼容而无法编译；此问题已在迭代 3 改为官方 API，并通过配置重启实现兼容。
- 升级前官方 Xray-core 缺少 fork 的 `features/bandwidth`；此问题已在迭代 3 将限速适配移入 XNode dispatcher 后解决。
- 当前 fork 的 CI 原用 Go 1.25.7（低于 `go.mod` 的 1.26）、发布和升级指向 `cedar2025`。已把 CI 工具链、镜像命名空间、安装脚本和 `xbctl` 下载地址改为本 fork。
- fork 的 PR 目标分支是 `dev`，原 CI 未监听该分支；Docker 发布任务也会在 PR 上尝试写入。已补上 `dev` PR 测试，并限定镜像发布只在 push 上运行。
- Xray Trojan 在证书自动启用 TLS 时曾修改传入的面板节点对象；已改为复制对象后生成配置，避免影响缓存的面板状态。
- sing-box 默认的私网地址阻断规则会把 `127.0.0.1` 回落也阻断；已增加只匹配配置目标和 Trojan inbound 的优先直连规则，并用真实 TLS 回落请求复验。

### 升级与验证状态

- [x] 官方 sing-box `v1.14.4` 已完成适配和升级。
- [x] 官方 Xray-core `v1.260327.1-0.20261031132019-a4a3dff5b20c` 已完成适配和升级。
- [x] 依赖更新后通过本机 `go test ./...`、race 测试、`go vet ./...`、`go mod tidy -diff` 和 Linux amd64/arm64 交叉构建。
- [x] GitHub Actions 已完成 `v1.14.1` 标签及 `dev` 分支验证：全仓测试、fallback 集成测试、Linux amd64/arm64 构建和 Release 发布成功。
- [x] 安装脚本已使用本 fork `dev` Release 资产；`dev` Release 的 arm64 资产已更新。若下载仍收到 504，可稍后重试同一 URL，或下载版本标签资产验证网络链路。
- [ ] 使用真实 Xboard 面板导出脱敏节点配置，补充 API 响应契约样本并覆盖目前未出现的字段变体。
- [x] 香港落地机的公网 HTTP/1.1/h2 回落和有效 Trojan 凭据测试已通过，详情见迭代 4。
- [x] Xboard 节点端口更新后 fallback 保留已在香港实机验证。
- [x] 用户已用浏览器打开新节点端口并确认显示主站。

## 交接步骤

1. 阅读本文件、`config.yml.example` 和相关提交。
2. 确认当前 `go.mod` 的 `require` 与 `replace`，勿跳过 fork 差异直接删除替换。
3. 在独立分支升级一个内核，先编译和运行该内核测试，再运行全仓测试。
4. 用官方 Xboard 的实际 API 响应测试热更新，并在 Linux 节点验证 fallback 和代理连接。
5. 每次迭代更新本文的基线、结论、验证命令和未解决风险。

## 官方依据

- Xboard：<https://github.com/cedar2025/Xboard>
- xboard-node：<https://github.com/cedar2025/Xboard-Node>
- Xray fallback：<https://xtls.github.io/en/config/features/fallback.html>
- sing-box Trojan inbound：<https://sing-box.sagernet.org/configuration/inbound/trojan/>

## 迭代 3：上游内核与依赖升级（实现和 CI 完成，Linux 实机回归待执行）

- 当前开发分支：`codex/core-dependency-updates`；该提交已合入远端 `dev`，`main` 保留上一个正式基线。
- 目标内核：官方 sing-box `v1.14.4`；官方 Xray-core `v26.10.31`（Go 伪版本 `v1.260327.1-0.20261031132019-a4a3dff5b20c`）。fork 替换已移除。
- Xray fork 的 bandwidth feature 已移到 XNode dispatcher：保留原始 Reader，在 Writer 上使用按用户共享的 limiter；同时适配上游 Xray UUID、Shadowsocks cipher API。
- sing-box fork 的运行时 `UpdateRules`、`UpdatableInbound` 和 InboundManager 移除/关闭 API 已改为官方接口；配置和用户变更执行整盒重启，启动失败时恢复旧配置。限速/设备限制元数据在仅限制变化时仍刷新。
- Go 依赖已通过 `go get -u ./...` 更新并 tidy；Vultr libdns 驱动与 govultr 最新版本不兼容，暂时固定 govultr `v3.20.0` 以保留 DNS API 行为。
- CI 和 Docker builder Go 工具链改为 1.27。
- 已通过：`go test ./...`、`go test -race -count=1 ./internal/...`、`go mod tidy -diff`，以及 Linux amd64/arm64 构建。GitHub Actions 的 tag/dev 全仓测试、fallback 集成测试、两架构构建及 Release 均成功。
- Xray Trojan TLS fallback 集成测试通过：HTTP/1.1 与 h2 请求回落到 backend；有效 Trojan 用户请求也能到达 backend。测试明确放行 Xray freedom outbound 的 `127.0.0.0/8`，因为 Xray 默认会黑洞 Trojan 访问私网地址；这只在集成测试配置中设置，不改变生产默认策略。
- 当前主机为 macOS，sing-box 完整集成测试启动时被系统 DNS 监听注册拒绝（`watch DNS configuration ... status 9`）。GitHub Linux CI 上两内核 fallback 集成测试已通过。剩余工作是目标 Linux 落地机的真实部署回归。
- 上游稳定版本核对：官方 Xray-core `v26.10.31`、sing-box `v1.14.4`，Go `1.27.3`；sing-box `1.15.0-alpha.13` 是预发布版本，暂不作为生产升级目标。

## 迭代 4：香港落地机 fallback 实测（2026-11-11）

- 目标机运行 XNode `7abd6d5`（`v1.14.1`），内核为 sing-box，Trojan inbound 初始端口 `10121`。
- 发现 Docker fallback 容器已运行，但 `/etc/xboard-node/config.yml` 的实例 `kernel.fallbacks` 为空，XNode 与 Nginx 回源原先未接通。已备份配置并设置默认回落 `127.0.0.1:8080`、ALPN `h2` 回落 `127.0.0.1:8082`；配置热加载/重启成功，健康检查返回 200。
- 从本地公网客户端请求 `https://gcphkb.dodoasclothing.uk:10121/`：HTTP/1.1 返回主站 200，HTTP/2 返回主站 200；两次都直连该落地机公网 IP。
- 使用当前 Xboard 用户 UUID 作为 Trojan 凭据进行标准握手和 TCP 转发，目标站返回 301，确认有效 Trojan 用户仍可代理连接。
- 2026-11-11，用户在 Xboard 将节点端口改为 `10221`；XNode 拉取配置后监听更新，新端口的 HTTP/1.1 和 h2 回落仍返回 200，当前面板用户的 Trojan 转发仍成功。错误密码测试返回 400；该测试不是普通 HTTPS 探测请求，普通 HTTPS 探测已返回主站 200。
- 用户自用的一键安装工具不属于本仓库发布内容；README 仅记录 XNode 本地 fallback 的配置契约和部署要求。
- 尚待：在干净 Linux 主机上完成一次端到端部署回归。

## 收尾审阅（2026-11-11）

- README 中英文版已说明 fallback 的适用协议、node/machine 模式与内核类型关系、服务监听要求、机器模式继承行为和健康检查端口限制。
- 静态安全审阅发现的加固项：health check 当前绑定所有网卡且无认证，部署时应在防火墙限制来源；面板 API token 会依 Xboard API 契约出现在请求参数中，面板连接应使用 HTTPS 并避免在代理/访问日志记录敏感 query；Docker 现存安装不会由本地 fallback 工具自动升级，应持续修补宿主机 Docker。
- fallback 只处理匹配的 Trojan TCP+TLS TLS 握手并提供普通网站回落，不能承诺抵御所有主动探测或封禁。
- 本环境没有 Go 工具链，未运行 `govulncheck`；发布前可在 Linux CI 增加依赖漏洞扫描，并持续核对 Xray、sing-box 和 Docker 的安全公告。
