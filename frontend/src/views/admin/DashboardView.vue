<template>
  <AppLayout>
    <!-- fork：管理员仪表盘整页换成 TokenRouter 的布局（components/admin/dashboard/fork/AdminDashboardPage.vue）；
         取数接口不变，上游的 ModelDistributionChart / TokenUsageTrend 留给用量页 -->
    <AdminDashboardPage />
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminDashboardPage from '@/components/admin/dashboard/fork/AdminDashboardPage.vue' // fork：TokenRouter 风格管理员仪表盘
</script>
