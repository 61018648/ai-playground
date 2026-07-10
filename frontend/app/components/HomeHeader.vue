<script setup lang="ts">
const search = ref('')
const authOpen = ref(false)
const userMenuOpen = ref(false)
const paymentOpen = ref(false)
const paymentMode = ref<'membership' | 'credits'>('membership')
const { siteName } = useSiteConfig()
const { toggle } = useSidebar()
const auth = useAuth()
const message = useMessage()

const displayName = computed(() => auth.user.value?.nickname || auth.user.value?.email?.split('@')[0] || '用户')
const shortUID = computed(() => auth.user.value?.id?.replace(/-/g, '').slice(0, 10) || '')
const membershipText = computed(() => ({
  free: 'FREE',
  v1: 'V1',
  v2: 'V2'
}[auth.user.value?.membershipLevel || 'free']))
const membershipName = computed(() => ({
  free: '免费会员',
  v1: 'V1会员',
  v2: 'V2会员'
}[auth.user.value?.membershipLevel || 'free']))
const avatarText = computed(() => displayName.value.slice(0, 1).toUpperCase())
const isAdmin = computed(() => auth.user.value?.role === 'admin')
const menuItems = computed(() => [
  { label: '个人中心', icon: 'i-lucide-user', to: '/profile' },
  ...(isAdmin.value ? [{ label: '管理后台', icon: 'i-lucide-layout-dashboard', to: '/admin' }] : []),
  { label: '我的收藏', icon: 'i-lucide-heart', to: '/media?tab=favorites' },
  { label: '加入用户交流群', icon: 'i-lucide-users', to: '/assistant' }
])

onMounted(() => {
  auth.loadMe()
})

const copyUID = async () => {
  if (!shortUID.value) return
  try {
    await navigator.clipboard.writeText(shortUID.value)
    message.success('复制成功', 'UID 已复制到剪贴板')
  } catch {
    message.error('复制失败', '请手动复制 UID')
  }
}

const logout = () => {
  userMenuOpen.value = false
  auth.logout()
}

const openPayment = (mode: 'membership' | 'credits') => {
  paymentMode.value = mode
  paymentOpen.value = true
  userMenuOpen.value = false
}
</script>

<template>
  <header class="flex items-center gap-2 sm:gap-4 lg:gap-6 px-4 sm:px-6 h-16 shrink-0 border-b border-default bg-default/80 backdrop-blur sticky top-0 z-30">
    <!-- 移动端汉堡按钮 -->
    <UButton
      icon="i-lucide-menu"
      color="neutral"
      variant="ghost"
      size="sm"
      class="lg:hidden shrink-0"
      aria-label="打开菜单"
      @click="toggle"
    />

    <!-- 标题区(小屏隐藏副标题) -->
    <div class="shrink-0 leading-tight">
      <p class="text-sm font-semibold text-highlighted whitespace-nowrap">
        {{ siteName }} 首页
      </p>
      <p class="hidden sm:block text-xs text-dimmed">
        AI 创作广场
      </p>
    </div>

    <!-- 搜索框(超小屏隐藏,改为图标按钮) -->
    <div class="flex-1 max-w-2xl hidden sm:block">
      <UInput
        v-model="search"
        icon="i-lucide-search"
        size="lg"
        placeholder="搜索模型、作品、提示词"
        :ui="{ root: 'w-full', base: 'rounded-full bg-elevated/60' }"
      />
    </div>

    <!-- 右侧操作 -->
    <div class="flex items-center gap-1.5 sm:gap-2 lg:gap-3 ml-auto shrink-0">
      <!-- 超小屏的搜索图标 -->
      <UButton
        icon="i-lucide-search"
        color="neutral"
        variant="ghost"
        size="sm"
        class="sm:hidden"
        aria-label="搜索"
      />

      <UButton
        color="warning"
        variant="solid"
        size="sm"
        icon="i-lucide-crown"
        class="rounded-full font-semibold"
        @click="openPayment('membership')"
      >
        <span class="hidden sm:inline">VIP 升级会员</span>
      </UButton>
      <UColorModeButton />
      <UPopover
        v-if="auth.user.value"
        v-model:open="userMenuOpen"
        :content="{ align: 'end', sideOffset: 10 }"
        :ui="{ content: 'p-0 rounded-xl shadow-xl ring-1 ring-default overflow-hidden bg-default w-[min(19rem,calc(100vw-1.5rem))]' }"
      >
        <UButton
          color="neutral"
          variant="ghost"
          size="sm"
          class="rounded-full p-0 h-9 w-9 overflow-hidden ring-1 ring-default transition-all duration-200 hover:ring-2 hover:ring-primary/60 hover:scale-105"
          aria-label="打开用户菜单"
        >
          <img
            v-if="auth.user.value.avatarUrl"
            :src="auth.user.value.avatarUrl"
            :alt="displayName"
            class="h-full w-full object-cover"
          >
          <span
            v-else
            class="h-full w-full grid place-items-center bg-gradient-to-br from-sky-100 via-indigo-100 to-amber-100 text-sm font-bold text-primary"
          >
            {{ avatarText }}
          </span>
        </UButton>

        <template #content>
          <div>
            <div class="p-5 pb-4 flex items-center gap-3">
              <div class="h-12 w-12 rounded-xl overflow-hidden ring-1 ring-default shrink-0 bg-elevated">
                <img
                  v-if="auth.user.value.avatarUrl"
                  :src="auth.user.value.avatarUrl"
                  :alt="displayName"
                  class="h-full w-full object-cover"
                >
                <div
                  v-else
                  class="h-full w-full grid place-items-center bg-gradient-to-br from-sky-100 via-indigo-100 to-amber-100 text-lg font-bold text-primary"
                >
                  {{ avatarText }}
                </div>
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <p class="font-bold text-highlighted truncate">
                    {{ displayName }}
                  </p>
                  <span class="shrink-0 rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-bold text-dimmed">
                    {{ membershipText }}
                  </span>
                </div>
                <div class="mt-1 flex items-center gap-1 text-xs text-dimmed">
                  <span>UID: {{ shortUID }}</span>
                  <button
                    type="button"
                    class="inline-flex h-5 w-5 items-center justify-center rounded-md hover:bg-elevated hover:text-highlighted"
                    aria-label="复制 UID"
                    @click="copyUID"
                  >
                    <UIcon
                      name="i-lucide-copy"
                      class="h-3.5 w-3.5"
                    />
                  </button>
                </div>
              </div>
            </div>

            <div class="flex items-center justify-between bg-muted/60 px-5 py-3 text-xs">
              <span class="text-muted font-medium">{{ membershipName }}</span>
              <button
                type="button"
                class="font-semibold text-fuchsia-500 hover:text-fuchsia-600"
                @click="openPayment('membership')"
              >
                查看会员权益
              </button>
            </div>

            <div class="px-4 py-3">
              <NuxtLink
                v-for="item in menuItems"
                :key="item.label"
                :to="item.to"
                class="flex h-11 items-center gap-3 rounded-lg px-2 text-sm font-medium text-toned hover:bg-elevated hover:text-highlighted"
                @click="userMenuOpen = false"
              >
                <UIcon
                  :name="item.icon"
                  class="h-4.5 w-4.5 text-dimmed"
                />
                <span class="flex-1">{{ item.label }}</span>
                <UIcon
                  name="i-lucide-chevron-right"
                  class="h-4 w-4 text-dimmed"
                />
              </NuxtLink>

              <div class="my-2 border-t border-default" />

              <button
                type="button"
                class="flex h-11 w-full items-center gap-3 rounded-lg px-2 text-sm font-medium text-toned hover:bg-elevated hover:text-highlighted"
                @click="logout"
              >
                <UIcon
                  name="i-lucide-log-out"
                  class="h-4.5 w-4.5 text-dimmed"
                />
                <span>退出登录</span>
              </button>
            </div>
          </div>
        </template>
      </UPopover>
      <UButton
        v-else
        color="primary"
        variant="solid"
        size="sm"
        class="rounded-full font-medium whitespace-nowrap"
        @click="authOpen = true"
      >
        <span class="hidden sm:inline">登录 / 注册</span>
        <span class="sm:hidden">登录</span>
      </UButton>
    </div>

    <AuthModal v-model:open="authOpen" />
    <PaymentModal
      v-model:open="paymentOpen"
      :initial-mode="paymentMode"
    />
  </header>
</template>
