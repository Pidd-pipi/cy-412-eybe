<template>
  <article class="card repair-card" :class="{ 'is-cancelled': repair.status === REPAIR_STATUS.CANCELLED }">
    <div class="repair-main">
      <div>
        <h3>{{ repair.title }}</h3>
        <p>{{ repair.description }}</p>
        <small>{{ repair.type }} · {{ repair.user?.nickname || '业主' }} · {{ new Date(repair.created_at).toLocaleString() }}</small>
      </div>
      <div class="repair-side">
        <RepairStatusBadge :status="repair.status" />
        <el-button v-if="canCancel" link type="danger" :loading="cancelling" @click="onCancel">撤销工单</el-button>
        <el-button link type="primary" @click="expanded = !expanded">{{ expanded ? '收起详情' : '查看详情' }}</el-button>
      </div>
    </div>
    <div v-if="expanded" class="repair-detail">
      <el-descriptions :column="2" size="small" border>
        <el-descriptions-item label="工单编号">#{{ repair.id }}</el-descriptions-item>
        <el-descriptions-item label="状态"><RepairStatusBadge :status="repair.status" /></el-descriptions-item>
        <el-descriptions-item label="报修类型">{{ repair.type }}</el-descriptions-item>
        <el-descriptions-item label="报修人">{{ repair.user?.nickname || '业主' }}（{{ repair.user?.phone || '-' }}）</el-descriptions-item>
        <el-descriptions-item label="处理人">{{ repair.handler ? repair.handler.nickname : '尚未分派' }}</el-descriptions-item>
        <el-descriptions-item label="评价">{{ repair.rating ? `${repair.rating} 星` : '暂无' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ new Date(repair.created_at).toLocaleString() }}</el-descriptions-item>
        <el-descriptions-item label="最近更新">{{ new Date(repair.updated_at).toLocaleString() }}</el-descriptions-item>
        <el-descriptions-item label="问题描述" :span="2">{{ repair.description }}</el-descriptions-item>
        <el-descriptions-item v-if="repair.status === REPAIR_STATUS.CANCELLED" label="撤销说明" :span="2">
          工单已由业主本人撤销，记录保留，不再计入待处理数量。
        </el-descriptions-item>
      </el-descriptions>
    </div>
  </article>
</template>
<script setup lang="ts">
import {computed, ref} from 'vue';
import {ElMessage, ElMessageBox} from 'element-plus';
import type {Repair} from '../../types';
import {REPAIR_STATUS, canOwnerCancel} from '../../constants/repair';
import {cancelRepair} from '../../api/repair';
import {useAuth} from '../../hooks/useAuth';
import RepairStatusBadge from './RepairStatusBadge.vue';

const props = defineProps<{ repair: Repair }>();
const emit = defineEmits<{ (e: 'cancelled', repair: Repair): void }>();
const {user} = useAuth();
const expanded = ref(false);
const cancelling = ref(false);

// 只有工单创建者本人在待受理/已分派阶段可见撤销按钮；物业与管理员不显示。
const canCancel = computed(() =>
  !!user.value && user.value.id === props.repair.user_id && canOwnerCancel(props.repair.status),
);

async function onCancel() {
  let reason = '';
  try {
    const {value} = await ElMessageBox.prompt('撤销后工单将标记为“已取消”，且不再计入物业待处理数量。', '确认撤销该报修工单？', {
      confirmButtonText: '确认撤销',
      cancelButtonText: '再想想',
      inputPlaceholder: '可填写撤销原因（选填）',
      inputValue: '',
      type: 'warning',
    });
    reason = value || '';
  } catch {
    return; // 用户放弃操作
  }
  cancelling.value = true;
  try {
    // 服务端再次校验创建者身份与状态机，越权/重复/已推进会返回明确原因。
    const updated = await cancelRepair(props.repair.id, reason);
    ElMessage.success('报修工单已撤销');
    emit('cancelled', updated);
  } catch (e) {
    // 直接回显后端给出的明确原因（重复取消、越权、状态已推进）。
    ElMessage.error((e as Error).message);
  } finally {
    cancelling.value = false;
  }
}
</script>
