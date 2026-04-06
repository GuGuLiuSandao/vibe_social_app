export default function GroupCreateModal({
  open,
  form,
  selectedMemberIds,
  friendCandidates,
  onChangeForm,
  onToggleMember,
  onClose,
  onSubmit,
  DiscordButton,
  DiscordSecondaryButton,
  DiscordInput,
  getAvatarColor,
  getInitials,
  toIdString,
  joinModePrivate,
  joinModeApproval,
  joinModePublic,
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/65 p-4">
      <div className="w-full max-w-4xl rounded-2xl border border-border bg-card p-6 shadow-glow">
        <h3 className="font-display text-xl font-bold text-foreground">创建群聊</h3>
        <p className="mt-1.5 text-sm text-muted-foreground">填写群信息，并从右侧好友列表中勾选成员。</p>
        <div className="mt-5 grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(320px,360px)]">
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-xs font-semibold text-foreground">群聊名称</label>
              <DiscordInput value={form.name} onChange={(event) => onChangeForm({ name: event.target.value })} placeholder="例如 产品讨论组" />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-semibold text-foreground">群头像 URL（可选）</label>
              <DiscordInput value={form.avatar} onChange={(event) => onChangeForm({ avatar: event.target.value })} placeholder="https://..." />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-semibold text-foreground">群简介</label>
              <textarea value={form.description} onChange={(event) => onChangeForm({ description: event.target.value })} placeholder="介绍一下这个群的定位和规则" className="min-h-[110px] w-full resize-none rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:ring-2 focus:ring-ring" />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-semibold text-foreground">加入方式</label>
              <select value={form.joinMode} onChange={(event) => onChangeForm({ joinMode: Number(event.target.value) })} className="h-10 w-full rounded-lg border border-input bg-background px-3 text-sm text-foreground outline-none focus:ring-2 focus:ring-ring">
                <option value={joinModePrivate}>私密群</option>
                <option value={joinModeApproval}>申请加入群</option>
                <option value={joinModePublic}>公开群</option>
              </select>
            </div>
            <div className="rounded-lg border border-border bg-muted px-3 py-2.5 text-xs text-muted-foreground">需选择至少 2 位好友（不包含自己）才能创建群聊。</div>
          </div>
          <div className="min-h-0 rounded-lg border border-border bg-background">
            <div className="flex items-center justify-between border-b border-border px-3 py-2.5">
              <label className="block text-xs font-semibold text-foreground">群成员（仅好友）</label>
              <span className="text-[11px] text-muted-foreground">已选 {selectedMemberIds.length} 人</span>
            </div>
            <div className="max-h-80 overflow-y-auto p-1">
              {friendCandidates.length === 0 ? (
                <p className="px-3 py-4 text-xs text-muted-foreground">暂无可邀请好友，请先与对方互相关注。</p>
              ) : (
                friendCandidates.map((member) => {
                  const memberId = toIdString(member.id);
                  const selected = selectedMemberIds.includes(memberId);
                  const memberName = member.nickname || member.username || `UID ${memberId}`;
                  return (
                    <label key={memberId} className={`mb-1 flex cursor-pointer items-center gap-2 rounded-md border border-transparent px-3 py-2 transition last:mb-0 ${selected ? "bg-secondary ring-1 ring-primary" : "hover:bg-muted"}`}>
                      <input type="checkbox" checked={selected} onChange={() => onToggleMember(memberId)} className="h-4 w-4 accent-[#5865f2]" />
                      <div className="flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full text-[10px] font-bold text-white" style={{ backgroundColor: member.avatar ? "transparent" : getAvatarColor(memberId) }}>
                        {member.avatar ? <img src={member.avatar} alt="avatar" className="h-full w-full object-cover" /> : getInitials(memberName)}
                      </div>
                      <div className="min-w-0">
                        <p className="truncate text-xs font-semibold text-foreground">{memberName}</p>
                        <p className="truncate text-[11px] text-muted-foreground">UID: {memberId}</p>
                      </div>
                    </label>
                  );
                })
              )}
            </div>
          </div>
        </div>
        <div className="mt-6 grid grid-cols-2 gap-2.5">
          <DiscordSecondaryButton onClick={onClose}>取消</DiscordSecondaryButton>
          <DiscordButton onClick={onSubmit}>创建群聊</DiscordButton>
        </div>
      </div>
    </div>
  );
}
