export default function GroupJoinRequestsCard({
  requests,
  pendingStatus,
  onApprove,
  onReject,
  DiscordButton,
  DiscordSecondaryButton,
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">入群申请 ({requests.length})</p>
      {requests.length === 0 ? (
        <p className="mt-3 text-xs text-muted-foreground">暂无申请</p>
      ) : (
        <div className="mt-3 max-h-72 space-y-2.5 overflow-y-auto">
          {requests.map((item) => (
            <div key={String(item.id)} className="rounded-lg border border-border bg-muted p-3">
              <p className="text-sm font-semibold text-foreground">{item.applicantNickname || item.applicantUsername || `UID ${String(item.applicantId)}`}</p>
              <p className="mt-1 text-[11px] text-muted-foreground">{item.message || "无附言"}</p>
              {Number(item.status) === pendingStatus ? (
                <div className="mt-3 flex gap-2">
                  <DiscordButton className="h-8 px-3 text-xs" onClick={() => onApprove(item.id)}>通过</DiscordButton>
                  <DiscordSecondaryButton className="h-8 px-3 text-xs" onClick={() => onReject(item.id)}>拒绝</DiscordSecondaryButton>
                </div>
              ) : (
                <p className="mt-3 text-[11px] text-muted-foreground">已处理</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
