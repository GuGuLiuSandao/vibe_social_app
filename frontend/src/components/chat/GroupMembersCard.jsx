export default function GroupMembersCard({
  members,
  currentUserId,
  activeGroupRole,
  groupRoleOwner,
  groupRoleAdmin,
  groupRoleMember,
  canManageGroup,
  canTransferOwnership,
  onSetAdmin,
  onUnsetAdmin,
  onTransferOwnership,
  onRemoveMember,
  DiscordSecondaryButton,
}) {
  return (
    <div className="rounded-md border border-[#3f4248] bg-[#232428] p-3">
      <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">成员操作 ({members.length})</p>
      <div className="mt-2 max-h-80 space-y-2 overflow-y-auto">
        {members.map((member) => {
          const memberId = String(member.userId);
          const isSelf = memberId === String(currentUserId || "");
          const memberRole = Number(member.role || 0);
          return (
            <div key={memberId} className="rounded-md border border-[#3a3d43] bg-[#1e1f22] p-2">
              <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-slate-100">{member.nickname || member.username || `UID ${memberId}`}</p>
                  <p className="truncate text-[11px] text-slate-400">UID {memberId} · {memberRole === groupRoleOwner ? "群主" : memberRole === groupRoleAdmin ? "管理员" : "普通成员"}</p>
                </div>
              </div>
              {canManageGroup && !isSelf ? (
                <div className="mt-2 flex flex-wrap gap-2">
                  {activeGroupRole === groupRoleOwner && memberRole === groupRoleMember ? <DiscordSecondaryButton className="!h-8 !px-3 !text-xs" onClick={() => onSetAdmin(memberId)}>设为管理员</DiscordSecondaryButton> : null}
                  {activeGroupRole === groupRoleOwner && memberRole === groupRoleAdmin ? <DiscordSecondaryButton className="!h-8 !px-3 !text-xs" onClick={() => onUnsetAdmin(memberId)}>取消管理员</DiscordSecondaryButton> : null}
                  {canTransferOwnership && memberRole !== groupRoleOwner ? <DiscordSecondaryButton className="!h-8 !px-3 !text-xs" onClick={() => onTransferOwnership(memberId)}>转让群主</DiscordSecondaryButton> : null}
                  {((activeGroupRole === groupRoleOwner && memberRole !== groupRoleOwner) || (activeGroupRole === groupRoleAdmin && memberRole === groupRoleMember)) ? <DiscordSecondaryButton className="!h-8 !border-[#5f2a33] !bg-[#3b1f24] !px-3 !text-xs !text-[#ffb4bf] hover:!bg-[#4a252d]" onClick={() => onRemoveMember(memberId)}>移出群聊</DiscordSecondaryButton> : null}
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
