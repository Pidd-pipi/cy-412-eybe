import {computed, type Ref} from 'vue';
import type {Repair} from '../types';

// 非终态状态集合：已完成、已关闭、已取消均不再计入待处理数量。
const CLOSED_STATUSES = ['done', 'closed', 'cancelled'];

export function useRepairStats(items: Ref<Repair[]>) {
  const counts = computed(() =>
    items.value.reduce<Record<string, number>>((a, v) => ((a[v.status] = (a[v.status] || 0) + 1), a), {}),
  );
  const openCount = computed(() =>
    items.value.filter((v) => !CLOSED_STATUSES.includes(v.status)).length,
  );
  return {counts, openCount};
}
