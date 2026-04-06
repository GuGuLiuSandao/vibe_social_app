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
    <div className="rounded-lg border border-border bg-card p-4">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">成员操作 ({members.length})</p>
      <div className="mt-3 max-h-80 space-y-2.5 overflow-y-auto">
        {members.map((member) => {
          const memberId = String(member.userId);
          const isSelf = memberId === String(currentUserId || "");
          const memberRole = Number(member.role || 0);
          return (
            <div key={memberId} className="rounded-lg border border-border bg-muted p-3">
              <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-foreground">{member.nickname || member.username || `UID ${memberId}`}</p>
                  <p className="truncate text-[11px] text-muted-foreground">UID {memberId} · {memberRole === groupRoleOwner ? "群主" : memberRole === groupRoleAdmin ? "管理员" : "普通成员"}</p>
                </div>
              </div>
              {canManageGroup && !isSelf ? (
                <div className="mt-3 flex flex-wrap gap-2">
                  {activeGroupRole === groupRoleOwner && memberRole === groupRoleMember ? <DiscordSecondaryButton className="h-8 px-3 text-xs" onClick={() => onSetAdmin(memberId)}>设为管理员</DiscordSecondaryButton> : null}
                  {activeGroupRole === groupRoleOwner && memberRole === groupRoleAdmin ? <DiscordSecondaryButton className="h-8 px-3 text-xs" onClick={() => onUnsetAdmin(memberId)}>取消管理员</DiscordSecondaryButton> : null}
                  {canTransferOwnership && memberRole !== groupRoleOwner ? <DiscordSecondaryButton className="h-8 px-3 text-xs" onClick={() => onTransferOwnership(memberId)}>转让群主</DiscordSecondaryButton> : null}
                  {((activeGroupRole === groupRoleOwner && memberRole !== groupRoleOwner) || (activeGroupRole === groupRoleAdmin && memberRole === groupRoleMember)) ? <DiscordSecondaryButton className="h-8 border-destructive bg-destructive px-3 text-xs text-destructive-foreground hover:bg-destructive" onClick={() => onRemoveMember(memberId)}>移出群聊</DiscordSecondaryButton> : null}
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
