# 功能文档目录说明

`docs/features/` 用于按需求维度归档完整的功能交付记录。

## 目录规则
- 每个独立需求/功能建立一个单独子目录
- 目录名建议使用短横线英文或拼音语义化命名，例如：
  - `block-system-v1/`
  - `community-management-v1/`
  - `moments-feed-v1/`

## 每个功能目录必备文件
- `requirement.md`：需求设计文档
- `technical-solution.md`：技术方案文档
- `code-review.md`：Code Review 意见与处理记录

## 标准交付顺序
1. 先写 `requirement.md`
2. 再写 `technical-solution.md`
3. 再进行功能开发
4. 开发完成后补充 `code-review.md`
5. 提交 commit 后，在 `technical-solution.md` 末尾追加 commit hash 与 commit message

## 模板
- 可参考 `docs/features/_template/`
