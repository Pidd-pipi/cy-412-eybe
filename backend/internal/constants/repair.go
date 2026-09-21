package constants

const (
	RepairStatusPending    = "pending"
	RepairStatusAssigned   = "assigned"
	RepairStatusProcessing = "processing"
	RepairStatusDone       = "done"
	RepairStatusClosed     = "closed"
	RepairStatusCancelled  = "cancelled"
)

var ValidRepairStatuses = map[string]bool{RepairStatusPending: true, RepairStatusAssigned: true, RepairStatusProcessing: true, RepairStatusDone: true, RepairStatusClosed: true, RepairStatusCancelled: true}

// RepairStatusCancellable 业主可自行撤销工单的状态集合：待受理、已分派。
// 处理中及之后的工单只能走完成/关闭流程，不允许业主撤销。
var RepairStatusCancellable = map[string]bool{RepairStatusPending: true, RepairStatusAssigned: true}
