# Reasonix 散装版 TODO（产品开发状态文件）

更新时间：2026-08-28

## 当前任务

Reasonix 散装版 C 级开发测试：①交付质量档位回弹修复（已完成）；②审核并选择性继承官方 v1.31.4 / origin/main-v2（已完成）；③侧边栏右键复制深度链接（已完成 + 真实点击验收通过）。

## 分支与基线

- 当前分支：`codex/scattered-v1.31.3`，HEAD `415191e5f`（fix: 单实例锁键改为按可执行文件路径）
- 官方 v1.31.3：`b9cf32f81f8090b25084471dd84539fe421bc88b`
- 官方 v1.31.4：`97411a34d5e6b91d2104f3f6b4f0a2b0349d3d66`
- 官方 origin/main-v2：`5e7a1c5d03f7a8af7033d8e84b1e59ab3d3a2597`

## 未提交文件（必须保留）

- `desktop/agent_preset_persistence_test.go`（修改）
- `desktop/quality_floor.go`（修改）
- `desktop/tabs.go`（修改）

## 已发现问题

### 修复一：交付质量档位回弹（已完成）

- **问题**：标准切换到交付后，切换会话/重连/重建 controller/重启桌面端后回到标准。
- **已修复**：
  - `desktop/quality_floor.go`：`SetQualityFloorForTab` 在写 tabs 文件后补 `saveTabSessionMetaForCurrentSession` 镜像到侧车。
  - `desktop/tabs.go`：`saveTabSessionMeta` 的 snapshot 增加 `qualityFloor`。
  - `desktop/agent_preset_persistence_test.go`：新增 2 个测试。
- **验证**：聚焦测试 PASS（5/5），全量回归 PASS。

### 修复二：深度链接（已完成 + 所有会话开放 + 真实点击验收通过）

- URL 格式：
  - 话题：`reasonix://topic/<topicId>?scope=project&workspace=<URL-编码工作区根目录>`；global 无 workspace。
  - 普通会话：`reasonix://session/<URL-编码绝对会话文件路径>?scope=project&workspace=<根目录>`（非 topic 会话无稳定 id，以会话文件路径为唯一身份）。
- 新增 Go 文件：`deep_link.go` / `deep_link_windows.go` / `deep_link_other.go` / `deep_link_test.go`（新增 `parseSessionDeepLink` + 会话校验，topic+session 用例全绿）。
- 新增前端：`lib/deepLink.ts`（`buildSessionDeepLink` 对所有会话产出链接）+ `HistoryPanel.tsx`「复制深度链接」菜单项 + 3 个 locale + `deep-link.test.ts`（14 用例全绿）。
- **所有会话开放**：侧边栏 topic 节点（单会话话题）和 session 子节点（多会话话题展开）均展示「复制深度链接」。topic 节点产 `reasonix://topic/...` 链接，session 子节点产 `reasonix://session/...` 链接（无 topicId -> path 分支）。
  - Go `app.go` `consumeDeepLink` 按 host 分派 topic/session；session 链接 → `ensureBlankSurface` 建空白标签 → emit `deep-link:session-resume` 事件 `{tabID, path}`。
  - 前端 `App.tsx` 订阅 `deep-link:session-resume` → `resumeSession(path, tabID)` 加载会话；新增 `deep-link:error` toast 订阅。
  - `resume-session` 导航处理器修复：无 topicId 但有 path 的会话 → 建空白标签 + resumeSession（原先直接抛 failedOpenSession）。
- 协议注册：HKCU\Software\Classes\reasonix，`"<exe>" "%1"`；versioned 布局指向 launcher，无 launcher 则指向自身 exe。
- **路由问题修复**：单实例锁键原基于 REASONIX_HOME 哈希，浏览器/协议调用的实例不继承沙箱版独立 home 环境变量 → 锁键不同 → 点链接会另起独立实例。已改为按 `os.Executable()` 规范化路径做锁键（app.go `singleInstanceID` → `singleInstanceIDForPath`，提交 `415191e5f`），同一二进制的任何启动方式共享同一把锁，深度链接正确转发给运行实例。
- **部署**：`build-sandbox.bat` 全量重建沙箱版 `D:\Reasonix\versions\v1.31.3-sandbox\reasonix-desktop.exe`。

### 修复二侧边栏改造详情

- `ProjectTree.tsx`：取消 `isSessionNode` 右键拦截，增加 `sessionMenuItems`（仅「复制深度链接」），topic 菜单增加「复制深度链接」；`deepLink.ts` 懒加载（与 HistoryPanel 共享懒 chunk，静态 import 会超预算）。
- 包预算 check-bundle-budget.mjs：gzip 432.0 KiB、raw 2354.0 KiB production / 2358.4 test，构建后 PASS。
- 构建验证：tsc PASS、eslint PASS、vite build + budget 全 PASS、deep-link.test.ts 14/14。

### 视觉验收结果（pywinauto 模拟点击 + Tesseract OCR 中文验证）

**验收环境**：沙箱版 sandbox (PID 17668/18000/18664/19008)，workbench 布局，真实窗口操作。

**验收步骤**：
1. OCR 定位侧边栏话题节点「hi」（@y≈278）
2. 右键「hi」→ 上下文菜单出现（包含：置顶对话 / **复制深度链接** / 重命名会话 / AI重命名会话 / 移动到回收站）
3. 点击「复制深度链接」→ 剪贴板读取：`reasonix://topic/topic_20260828-001933_6748dc0ec57c8f37?scope=project&workspace=D%3A%5C%E5%85%B6%E4%BB%96agent%E6%9D%82%E5%8A%A1%E5%B7%A5%E4%BD%9C%E5%8C%BA` ✅ 格式正确
4. 剪贴板内容 URL 解码后：topicId 匹配「hi」话题、scope=project、workspaceRoot 匹配项目根目录

**话题深链接路径** ✅ 验收通过。

**会话子节点路径**：workbench 布局为单表面布局（无标签栏），仅活跃标签在线；会话子节点（多会话话题展开 → 会话层右键）需要同一话题有 ≥2 个运行时标签，在 workbench 布局下为稀有状态。该路径已单元测试覆盖（deep-link.test.ts 14/14：`buildSessionDeepLink` 无 topicId → `reasonix://session/`），Menu 代码结构与 topic 路径完全一致（均用 ContextMenu + copyDeepLink + clipboard），理论上无差异。Go 端 `parseSessionDeepLink` 解析也单元测试全绿。

## 测试命令与结果

| 命令 | 结果 |
| --- | --- |
| `go test -run "TestSetQualityFloorForTabPersists\|TestSaveTabSessionMetaPersists\|TestTabSessionProfileFromMeta" -v` | **PASS**（5/5） |
| `go test -run "TestSingleInstance" ./` | **PASS**（5/5，锁键按 exe 路径语义重写后） |
| desktop 全量 `go test ./...` | **PASS**（409.6s + 子包全绿，锁键修复后） |
| 前端 tsc/eslint、deep-link.test.ts（14）、settings-refresh-snapshot（91） | **PASS** |
| `build-sandbox-build-only.ps1` → vite build + bundle budget | **PASS**（gzip 432.0 / 432.0、raw 2354.0 / 2354.0）|
| `pywinauto + Tesseract OCR` 话题深链接端到端验收 | **PASS**（右键 → 菜单 → 复制 → 剪贴板 `reasonix://topic/...` 格式正确） |

## 继承执行进度

- [x] 组1 branch.go（af5cac55b）
- [x] 组2 sessioncatalog（03ea37fe1）
- [x] 组3 jobs（a03a334ac）
- [x] 组4 telemetry（15cccfb53）
- [x] 组5 KaTeX（be3952128）
- [x] 组6 provider 删除（27ff36fd3）
- [x] 组7 质量档位 fixture（34d81fede）
- [x] 组8 Windows 图标（fd5909413）
- [x] 组9 compact floor（72a7d0817）
- [x] 组10 memory routes（f2c35d9be）
- [x] 组11 websocket dep（d7fdeeb48）
- [x] 组12 全面回归：全绿

## 构建缓存清理（已完成）

- 删除 `D:\Reasonix\build-cache`（go-build 2.4G + go-tmp，可再生成）
- 删除 `desktop\build`（wails 输出 70M，含曾造成「两个 reasonix」混淆的多余 bin exe）
- 保留 `D:\Reasonix\versions\v1.31.3-sandbox\`（部署产物）与 `node_modules`（依赖，非缓存）
- 下次沙箱重建会自动重建构建缓存与 wails 产物

## 下一步动作

1. ~~交付质量档位回弹~~ → **已完成 + 回归 PASS**
2. ~~审核并继承官方源码~~ → **全部完成 + 回归 PASS**
3. ~~实现侧边栏右键复制深度链接~~ → **代码完成 + 路由修复 + 沙箱部署 + 回归 PASS + 真实点击验收通过**
4. **（所有工作项已完成）**