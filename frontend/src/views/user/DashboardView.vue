<template>
  <AppLayout>
    <!-- fork：仪表盘整页换成 TokenRouter 的布局（components/user/dashboard/fork/UserDashboardPage.vue）；
         上游的 UserDashboardStats / Charts / RecentUsage 不再挂载，平台拆分与配额搬到 fork 的 UserDashboardPlatforms -->
    <UserDashboardPage />
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import UserDashboardPage from '@/components/user/dashboard/fork/UserDashboardPage.vue' // fork：TokenRouter 风格仪表盘
</script>
