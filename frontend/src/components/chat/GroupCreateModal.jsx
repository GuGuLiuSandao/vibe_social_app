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
      <div className="discord-surface w-full max-w-4xl rounded-xl p-5">
        <h3 className="font-display text-xl font-bold text-white">创建群聊</h3>
        <p className="mt-1 text-sm text-slate-400">填写群信息，并从右侧好友列表中勾选成员。</p>
        <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(320px,360px)]">
          <div className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-semibold text-slate-300">群聊名称</label>
              <DiscordInput value={form.name} onChange={(event) => onChangeForm({ name: event.target.value })} placeholder="例如 产品讨论组" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-semibold text-slate-300">群头像 URL（可选）</label>
              <DiscordInput value={form.avatar} onChange={(event) => onChangeForm({ avatar: event.target.value })} placeholder="https://..." />
            </div>
            <div>
              <label className="mb-1 block text-xs font-semibold text-slate-300">群简介</label>
              <textarea value={form.description} onChange={(event) => onChangeForm({ description: event.target.value })} placeholder="介绍一下这个群的定位和规则" className="min-h-[90px] w-full resize-none rounded-md border border-[#3f4248] bg-[#1e1f22] px-3 py-2 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-[#5865f2]" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-semibold text-slate-300">加入方式</label>
              <select value={form.joinMode} onChange={(event) => onChangeForm({ joinMode: Number(event.target.value) })} className="h-10 w-full rounded-md border border-[#3f4248] bg-[#1e1f22] px-3 text-sm text-slate-100 outline-none focus:border-[#5865f2]">
                <option value={joinModePrivate}>私密群</option>
                <option value={joinModeApproval}>申请加入群</option>
                <option value={joinModePublic}>公开群</option>
              </select>
            </div>
            <div className="rounded-md border border-[#3f4248] bg-[#232428] px-3 py-2 text-xs text-slate-400">需选择至少 2 位好友（不包含自己）才能创建群聊。</div>
          </div>
          <div className="min-h-0 rounded-md border border-[#3f4248] bg-[#1e1f22]">
            <div className="flex items-center justify-between border-b border-[#2a2c30] px-3 py-2">
              <label className="block text-xs font-semibold text-slate-300">群成员（仅好友）</label>
              <span className="text-[11px] text-slate-400">已选 {selectedMemberIds.length} 人</span>
            </div>
            <div className="max-h-72 overflow-y-auto">
              {friendCandidates.length === 0 ? (
                <p className="px-3 py-4 text-xs text-slate-400">暂无可邀请好友，请先与对方互相关注。</p>
              ) : (
                friendCandidates.map((member) => {
                  const memberId = toIdString(member.id);
                  const selected = selectedMemberIds.includes(memberId);
                  const memberName = member.nickname || member.username || `UID ${memberId}`;
                  return (
                    <label key={memberId} className={`flex cursor-pointer items-center gap-2 border-b border-[#2a2c30] px-3 py-2 transition last:border-b-0 ${selected ? "bg-[#2a315a]" : "hover:bg-[#2a2c30]"}`}>
                      <input type="checkbox" checked={selected} onChange={() => onToggleMember(memberId)} className="h-4 w-4 accent-[#5865f2]" />
                      <div className="flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full text-[10px] font-bold text-white" style={{ backgroundColor: member.avatar ? "transparent" : getAvatarColor(memberId) }}>
                        {member.avatar ? <img src={member.avatar} alt="avatar" className="h-full w-full object-cover" /> : getInitials(memberName)}
                      </div>
                      <div className="min-w-0">
                        <p className="truncate text-xs font-semibold text-slate-100">{memberName}</p>
                        <p className="truncate text-[11px] text-slate-400">UID: {memberId}</p>
                      </div>
                    </label>
                  );
                })
              )}
            </div>
          </div>
        </div>
        <div className="mt-4 grid grid-cols-2 gap-2">
          <DiscordSecondaryButton onClick={onClose}>取消</DiscordSecondaryButton>
          <DiscordButton onClick={onSubmit}>创建群聊</DiscordButton>
        </div>
      </div>
    </div>
  );
}
