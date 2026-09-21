import type {RepairStatus} from '../types';

export const REPAIR_STATUS:Record<Uppercase<RepairStatus>,RepairStatus>={PENDING:'pending',ASSIGNED:'assigned',PROCESSING:'processing',DONE:'done',CLOSED:'closed',CANCELLED:'cancelled'};

export const repairStatusText:Record<RepairStatus,string>={pending:'待受理',assigned:'已分派',processing:'处理中',done:'已完成',closed:'已关闭',cancelled:'已取消'};

// 业主可自行撤销工单的状态：待受理、已分派；处理中及之后只能走完成/关闭流程。
export const REPAIR_CANCELLABLE_STATUSES:RepairStatus[]=[REPAIR_STATUS.PENDING,REPAIR_STATUS.ASSIGNED];

export function canOwnerCancel(status:RepairStatus):boolean{return REPAIR_CANCELLABLE_STATUSES.includes(status)}
