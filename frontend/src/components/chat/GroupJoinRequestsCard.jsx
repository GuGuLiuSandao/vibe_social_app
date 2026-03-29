export default function GroupJoinRequestsCard({
  requests,
  pendingStatus,
  onApprove,
  onReject,
  DiscordButton,
  DiscordSecondaryButton,
}) {
  return (
    <div className="rounded-md border border-[#3f4248] bg-[#232428] p-3">
      <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">入群申请 ({requests.length})</p>
      {requests.length === 0 ? (
        <p className="mt-2 text-xs text-slate-500">暂无申请</p>
      ) : (
        <div className="mt-2 max-h-72 space-y-2 overflow-y-auto">
          {requests.map((item) => (
            <div key={String(item.id)} className="rounded-md border border-[#3a3d43] bg-[#1e1f22] p-2">
              <p className="text-sm font-semibold text-slate-100">{item.applicantNickname || item.applicantUsername || `UID ${String(item.applicantId)}`}</p>
              <p className="mt-1 text-[11px] text-slate-400">{item.message || "无附言"}</p>
              {Number(item.status) === pendingStatus ? (
                <div className="mt-2 flex gap-2">
                  <DiscordButton className="!h-8 !px-3 !text-xs" onClick={() => onApprove(item.id)}>通过</DiscordButton>
                  <DiscordSecondaryButton className="!h-8 !px-3 !text-xs" onClick={() => onReject(item.id)}>拒绝</DiscordSecondaryButton>
                </div>
              ) : (
                <p className="mt-2 text-[11px] text-slate-500">已处理</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
