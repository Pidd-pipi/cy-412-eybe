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

// OwnerCancellableStatuses 业主仅可在待受理或已分派阶段撤销本人工单，
// 处理中及之后必须走原有的完成/关闭流程。
var OwnerCancellableStatuses = map[string]bool{RepairStatusPending: true, RepairStatusAssigned: true}
