import {request} from '../utils/request';import type{Repair,RepairStatus}from'../types';

export const listRepairs=(status?:string)=>request<Repair[]>(`/repairs${status?`?status=${status}`:''}`);

export const createRepair=(data:Pick<Repair,'title'|'description'|'type'|'images'>)=>request<Repair>('/repairs',{method:'POST',body:JSON.stringify(data)});

export const assignRepair=(id:number,handler_id:number)=>request<Repair>(`/repairs/${id}/assign`,{method:'PATCH',body:JSON.stringify({handler_id})});

export const updateRepairStatus=(id:number,status:RepairStatus)=>request<Repair>(`/repairs/${id}/status`,{method:'PATCH',body:JSON.stringify({status})});

// 业主本人撤销报修工单：仅待受理/已分派可撤销；后端对越权、重复、状态推进分别返回明确原因。
export const cancelRepair=(id:number,reason='')=>request<Repair>(`/repairs/${id}/cancel`,{method:'POST',body:JSON.stringify({reason})});
