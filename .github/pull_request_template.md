## 这个 PR 改的是哪一层？（必选其一）
- [ ] **A 资产层**（`internal/canon/` schema、canon 表 YAML、`adr/`）→ 必须打标签 `asset-change` 并附 ADR
- [ ] **B 代码层**（`internal/` 其余包、`cmd/`、测试）→ 无需 ADR
- [ ] 文档 / CI

## 关联
Refs #  ·  issue #1 缺陷编号（P0-x / P1-x）：

## 验收命令（贴出你本地跑通的输出）
```
make check
```

## 检查表
- [ ] 测试先红后绿（贴出先失败的证据或说明为什么不适用）
- [ ] 新门禁带 issue #1 实际缺陷的复现用例（E5 渔捕快 / E9 A福 / E12 二次相识 / E30 过客有期…）
- [ ] 行为不变量用了 PBT（rapid），不只是 happy path
- [ ] 包依赖方向符合 docs/ARCHITECTURE.md（canon ◀ state ◀ gates ◀ passes）
- [ ] 未引入未报批的新依赖（已批：yaml.v3 / rapid）
- [ ] 对外接口（导出函数/CLI/报表格式）变更已写 CHANGELOG
