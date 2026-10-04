# 实习志愿分配工具

仅使用 Go 标准库实现的本地命令行工具：在学生志愿与项目接收顺序之间计算稳定分配，并可核查外部提交的分配方案。

## 构建与运行

```sh
go build -o matcher .
./matcher match input.json                  # 计算稳定分配
./matcher check input.json assignment.json  # 核查候选分配
```

`match` 始终输出 JSON 报告并以 0 退出；`check` 在方案不合法或不稳定时以 1 退出，输入文件错误以 2 退出。

## 输入格式

```json
{
  "students": [{"id": "s1", "preferences": ["p1", "p2"]}],
  "projects": [{"id": "p1", "capacity": 2, "preferences": ["s1", "s2"]}]
}
```

校验规则：标识非空且唯一；志愿/接收名单内不得重复、不得引用未知对象；容量为非负整数。只有学生名单包含项目且项目名单包含该学生时，双方才互相可接受，才可能匹配。

## 分配口径（match）

- 采用学生提案、项目暂留的延迟接受算法：学生按自己的偏好顺序依次申请；项目未满则暂留；满额时用更优学生替换当前最差的暂留者；被拒或被替换的学生继续申请下一志愿。
- 学生起始处理顺序为标识字典序；算法结果与输入数组的排列无关（测试覆盖）。
- 同一学生最多获配一个项目，项目不超容量；零容量项目正常出现在剩余名额中，未获配学生列入 `unmatchedStudents`。
- 每次提案产生一条记录：`held`（暂留）、`rejected`（拒绝，附原因）、`accepted-replacing`（替换接收，含 `replaced` 与指向被替换者暂留记录的 `replacedRecord` 序号）。最终 `assignments` 与提案记录链一致。

输出报告包含 `assignments`、`unmatchedStudents`、`remainingCapacity`、`proposals` 四部分，示例见 `examples/input.json`。

## 核查口径（check）

候选分配文件格式：`{"assignments": [{"student": "s1", "project": "p1"}]}`。

- 合法性：学生至多出现一次、项目不超容量、每对匹配双方互相可接受、不引用未知对象。
- 稳定性：若学生更偏好某项目（未获配视为低于任意可接受项目），且该项目有空位、或该项目更偏好该学生胜过其当前某位获配者，则构成阻塞对，全部列出于 `blockingPairs` 并附原因。

## 测试

```sh
go test ./...
```

覆盖替换链、名单不对称、零容量、输入顺序无关性、校验错误以及不稳定方案的阻塞对检测。
