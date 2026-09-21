<template>
  <section>
    <header class="page-head">
      <div>
        <p class="eyebrow">REPAIR CENTER</p>
        <h2>报修管理</h2>
      </div>
      <el-button type="primary" @click="dialog=true">提交报修</el-button>
    </header>
    <div class="toolbar">
      <el-select v-model="status" placeholder="全部状态" clearable style="width: 180px" @change="load">
        <el-option v-for="(t,k) in repairStatusText" :key="k" :label="t" :value="k"/>
      </el-select>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="repair-list">
      <RepairCard v-for="v in items" :key="v.id" :repair="v" @cancelled="load"/>
      <EmptyState v-if="!items.length"/>
    </div>
    <el-dialog v-model="dialog" title="提交报修工单">
      <el-form>
        <el-input v-model="form.title" placeholder="报修标题"/>
        <el-input v-model="form.description" type="textarea" placeholder="问题描述"/>
        <el-select v-model="form.type">
          <el-option label="水电" value="水电"/>
          <el-option label="家具" value="家具"/>
          <el-option label="公共设施" value="公共设施"/>
          <el-option label="其他" value="其他"/>
        </el-select>
      </el-form>
      <template #footer>
        <el-button @click="dialog=false">取消</el-button>
        <el-button type="primary" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </section>
</template>
<script setup lang="ts">
import {ref, onMounted} from 'vue';
import {ElMessage} from 'element-plus';
import {listRepairs, createRepair} from '../api/repair';
import {repairStatusText} from '../constants/repair';
import type {Repair} from '../types';
import RepairCard from '../components/common/RepairCard.vue';
import EmptyState from '../components/common/EmptyState.vue';

const items = ref<Repair[]>([]);
const status = ref('');
const dialog = ref(false);
const form = ref({title: '', description: '', type: '水电', images: ''});

// 拉取服务端最新列表：撤销后调用，完成“操作后刷新回读”，列表与详情状态同步。
async function load() {
  items.value = await listRepairs(status.value);
}

async function submit() {
  try {
    await createRepair(form.value);
    ElMessage.success('报修工单已提交');
    dialog.value = false;
    form.value = {title: '', description: '', type: '水电', images: ''};
    await load();
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}

onMounted(load);
</script>
