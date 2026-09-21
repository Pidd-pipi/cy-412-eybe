package constants

const (
	MessageOK              = "ok"
	MessageUnauthorized    = "登录已失效"
	MessageForbidden       = "无权限执行此操作"
	MessageValidation      = "请求参数不合法"
	MessageNotFound        = "资源不存在"
	MessagePaymentSuccess  = "支付宝沙箱支付成功"
	MessageRepairCreated   = "报修工单已提交"
	MessageRepairCancelled = "报修工单已撤销"
)

// 业主撤销工单失败时的明确原因文案（服务层按场景包装实体与角色信息）
const (
	ReasonRepairCancelForbidden  = "only the owner who created the repair can cancel it"
	ReasonRepairAlreadyCancelled = "repair is already cancelled"
	ReasonRepairNotCancellable   = "repair status does not allow cancellation, please use complete/close workflow"
	ReasonRepairStatusAdvanced   = "repair status has advanced and can no longer be cancelled"
	ReasonRepairCancelNotStaff   = "staff/admin cannot cancel a repair on behalf of the owner"
)
