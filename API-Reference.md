# Linkit API Reference

本文档描述当前后端可直接对接的上传相关接口。

## 1. 通用约定

### 1.1 基础信息
- Base URL：`http://<host>:3301`
- 内容类型：
  - JSON 接口：`application/json`
  - 上传接口：`multipart/form-data`

### 1.2 统一响应结构
所有接口返回统一结构：

```json
{
  "msg": "ok",
  "data": {},
  "code": 200
}
```

- `code=200` 表示业务成功
- 非 `200` 表示业务失败（即使 HTTP 状态码可能仍为 `200`）

### 1.3 认证方式（与上传相关）
- `/api/upload` 不强制登录。
- 已登录用户可通过 Cookie 会话上传。
- 第三方可通过请求头 `Authorization: {token}` 或 `Authorization: Bearer {token}` 识别身份。

---

## 2. 查询分片状态

### 2.1 接口
- 方法：`GET`
- 路径：`/api/upload`

### 2.2 Query 参数
| 参数名 | 类型 | 是否必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `uploadId` | `string` | 是 | 无 | 上传任务 ID |

### 2.3 成功响应体
以下字段均为 `data` 对象内字段。

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `uploaded` | `int64[]` | 是 | 已存在的分片索引列表；任务不存在时返回 `[]` |

---

## 3. 上传文件（直传/分片）

### 3.1 接口
- 方法：`POST`
- 路径：`/api/upload`
- Content-Type：`multipart/form-data`

### 3.2 Form 字段
| 参数名 | 类型 | 是否必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `file` | `file` | 是 | 无 | 文件内容，仅支持单文件（一次只能传 1 个文件） |
| `uploadId` | `string` | 否 | `{当前毫秒时间}-{上传文件名}` | 上传任务 ID，分片上传必须保持同一个值 |
| `filename` | `string` | 否 | `multipart` 文件头中的 `filename` | 文件名 |
| `filesize` | `int64` | 否 | `multipart` 文件头中的 `size` | 文件总大小（字节） |
| `chunkIndex` | `int64` | 否 | 无 | 当前分片索引（从 0 开始） |
| `totalChunks` | `int64` | 否 | 无 | 分片总数 |
| `chunkSize` | `int64` | 否 | 无 | 当前分片大小（仅回显用途） |

### 3.3 选填字段默认值（重点）
- `uploadId` 未传：自动生成  
  规则：`{当前毫秒时间}-{上传文件名}`
- `filename` 未传：使用上传文件头中的 `filename`
- `filesize` 未传：使用上传文件头中的 `size`

### 3.4 行为说明
- 服务端根据 `filesize` 与阈值判断是否需要分片上传。
- 当走分片上传时，`chunkIndex` 和 `totalChunks` 需要满足有效范围，否则返回 `code=400`（分片参数错误）。
- 同一个分片任务必须使用同一个 `uploadId`，否则无法正确合并。

### 3.5 成功响应体
以下字段均为 `data` 对象内字段。

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `merged` | `bool` | 是 | 是否已完成合并/入库 |
| `uploadId` | `string` | 是 | 上传任务 ID |
| `filename` | `string` | 是 | 文件名 |
| `size` | `int64` | 否 | 文件大小（仅在合并完成时返回） |
| `shareCode` | `string` | 否 | 分享码（仅在合并完成时返回） |
| `resourceId` | `int64` | 否 | 资源 ID（仅在合并完成时返回） |
| `skipped` | `bool` | 否 | 当前分片是否已存在（存在时返回 `true`） |
| `chunkIndex` | `int64` | 否 | 当前分片索引（分片流程返回） |
| `totalChunks` | `int64` | 否 | 分片总数（分片流程返回） |
| `chunkSize` | `int64` | 否 | 分片大小（分片流程回显） |
