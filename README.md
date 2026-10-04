# 实习志愿分配工具

纯 Go 标准库实现的本地命令行工具：在学生志愿与项目接收顺序之间计算稳定分配，
并可独立核查外部提交的分配方案。无网页、无第三方依赖。

## 构建

```sh
go build -o matching .
```

## 输入格式

```json
{
  "students": [{"id": "s1", "preferences": ["p1", "p2"]}],
  "projects": [{"id": "p1", "capacity": 1, "preferences": ["s1"]}]
}
```

- `id` 全局唯一（学生、项目各自命名空间内不得重复，不能为空）。
- `preferences` 按偏好从强到弱排列，不得重复，不得引用未知对象。
- `capacity` 为非负整数；`0` 表示该项目不接收任何人。
- 只有学生列出项目 **且** 项目列出学生，双方才允许匹配（单边列出无效）。

## 分配命令 `match`

```sh
./matching match examples/instance.json   # 或从 stdin：./matching match -
```

采用学生提案、项目暂留的延迟接受（deferred acceptance）过程：

1. 学生按标识字典序进入处理队列，依次向自己尚未申请过的最优志愿提案。
2. 项目有空位则暂留；满额时若项目更偏好新申请者，则替换当前最差的暂留者，
   否则拒绝。
3. 被拒绝或被替换的学生重新入队，继续申请下一志愿，直到获配或志愿耗尽。

输入对象（学生/项目）的排列顺序不影响最终结果。每名学生最多获配一个项目，
项目绝不超容量。

### 输出报告

- `assignments`：最终去向（按学生标识排序）。
- `unmatched_students`：未获配学生。
- `remaining_capacity`：各项目剩余名额。
- `records`：每次提案一条记录，`action` 为 `held`（暂留）、`rejected`
  （拒绝）或 `replaced`（替换），`reason` 说明依据；替换记录通过
  `replaced_record` 关联被替换暂留记录的 `seq`。回放所有记录
  （`held`/`replaced` 生效、被替换记录失效）可精确还原 `assignments`。

## 核查命令 `check`

```sh
./matching check examples/candidate.json
```

输入在实例基础上增加 `assignments`（`{"student", "project"}` 列表）。核查口径：

- **结构有效性**（`valid`/`errors`）：学生至多获配一次、标识均已知、
  双方互相列出（可接受性）、项目不超容量。
- **阻塞对**（`blocking_pairs`）：学生更偏好某项目（未获配视为低于任意可
  接受项目），且该项目有空位、或该项目更偏好该学生（相对某位现任获配者）。
- `stable` 当且仅当结构有效且不存在阻塞对。方案不稳定或无效时进程以
  退出码 1 结束，用法/输入错误以退出码 2 结束。

## 测试

```sh
go test ./...
```

覆盖替换链与记录对账、名单不对称（单边列出）、零容量与未获配、输入排列
不变性、非法输入校验，以及稳定/不稳定/超容量候选方案的核查。
