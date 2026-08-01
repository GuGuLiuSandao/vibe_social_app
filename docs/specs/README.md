# Product Behavior Specifications

本目录维护跨版本稳定的产品行为契约，为需求设计、测试设计和长期回归提供共同依据。

规格按业务域组织，每条行为使用稳定 ID，例如：

```text
AUTH-001
CHAT-001
RELATION-001
COMMUNITY-001
```

每条规格描述触发条件、输入、权限、系统行为、输出和错误行为。涉及对外行为的 Develop Loop 变更应更新对应规格，并让 Requirement Contract、Test Cases 和自动化测试引用相同 ID。

- [Authentication](authentication.md)
- [WebSocket](websocket.md)
- [Client identity and WebSocket builders](client-identity.md)
- `traceability.json` 是规格、Test Case 与自动化测试之间的机器可校验映射。
