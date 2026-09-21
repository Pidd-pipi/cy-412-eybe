<template><el-tag :type="kind" :effect="effect">{{repairStatusText[status]}}</el-tag></template>
<script setup lang="ts">
import {computed}from'vue';
import{repairStatusText,REPAIR_STATUS}from'../../constants/repair';
import type{RepairStatus}from'../../types';
const p=defineProps<{status:RepairStatus}>();
const kind=computed(()=>({pending:'warning',assigned:'primary',processing:'info',done:'success',closed:'danger',cancelled:'info'}[p.status] as 'success'));
const effect=computed(()=>(p.status===REPAIR_STATUS.CANCELLED?'plain':'light') as 'plain');
</script>
