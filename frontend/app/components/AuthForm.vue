<script setup lang="ts">
type Mode = 'login' | 'register' | 'forgot'
const mode = ref<Mode>('login')
const emit = defineEmits<{ success: [] }>()
const auth = useAuth()
const route = useRoute()

const email = ref('')
const password = ref('')
const remember = ref(true)
const showPassword = ref(false)

// 注册 / 找回密码 共用字段
const code = ref('')
const inviteCode = ref('')
const message = ref('')
const error = ref('')
const sendingCode = ref(false)
const submitting = ref(false)

// 验证码倒计时
const countdown = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  const code = route.query.invite || route.query.inviteCode
  if (typeof code === 'string' && code.trim()) {
    inviteCode.value = code.trim()
    mode.value = 'register'
  }
})

const canSendCode = computed(() => countdown.value === 0 && /\S+@\S+\.\S+/.test(email.value))

const codePurpose = computed(() => (mode.value === 'forgot' ? 'forgot_password' : 'register'))

const sendCode = async () => {
  if (!canSendCode.value) return
  sendingCode.value = true
  message.value = ''
  error.value = ''
  try {
    const res = await auth.sendCode(email.value, codePurpose.value)
    message.value = res.devCode ? `验证码已发送，开发环境验证码：${res.devCode}` : '验证码已发送'
    countdown.value = Math.min(res.expiresIn, 60)
    timer = setInterval(() => {
      countdown.value -= 1
      if (countdown.value <= 0 && timer) {
        clearInterval(timer)
        timer = null
      }
    }, 1000)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '验证码发送失败'
  } finally {
    sendingCode.value = false
  }
}

// 只保留数字、最多 6 位
watch(code, (val) => {
  const cleaned = val.replace(/\D/g, '').slice(0, 6)
  if (cleaned !== val) code.value = cleaned
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})

const heading = computed(() => ({
  login: '欢迎登录',
  register: '欢迎注册',
  forgot: '找回密码'
}[mode.value]))

const subtitle = computed(() => ({
  login: '使用邮箱账号安全登录',
  register: '使用邮箱账号快速注册',
  forgot: '通过邮箱验证码重置你的密码'
}[mode.value]))

const submitLabel = computed(() => ({
  login: '邮箱登录',
  register: '邮箱注册',
  forgot: '重置密码'
}[mode.value]))

const passwordLabel = computed(() => (mode.value === 'forgot' ? '新密码' : '密码'))
const needCode = computed(() => mode.value !== 'login')

const onSubmit = async () => {
  message.value = ''
  error.value = ''
  submitting.value = true
  try {
    if (mode.value === 'login') {
      await auth.login({ email: email.value, password: password.value, remember: remember.value })
      message.value = '登录成功'
      emit('success')
    } else if (mode.value === 'register') {
      await auth.register({
        email: email.value,
        password: password.value,
        code: code.value,
        inviteCode: inviteCode.value || undefined
      })
      message.value = '注册成功'
      emit('success')
    } else {
      await auth.forgotPassword({ email: email.value, code: code.value, newPassword: password.value })
      message.value = '密码已重置，请使用新密码登录'
      mode.value = 'login'
      password.value = ''
      code.value = ''
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col w-full md:w-[56%] p-8 sm:p-10">
    <!-- Logo -->
    <div class="flex items-center justify-center gap-2">
      <div class="flex items-center justify-center w-10 h-10 rounded-xl bg-gradient-to-br from-sky-400 to-blue-600 text-white shadow">
        <UIcon
          name="i-lucide-sparkles"
          class="w-5 h-5"
        />
      </div>
      <span class="text-2xl font-bold tracking-tight text-highlighted">季星<span class="text-primary">Ai</span></span>
    </div>

    <h1 class="mt-5 text-xl font-bold text-center text-highlighted">
      {{ heading }}
    </h1>
    <p class="mt-1 text-sm text-center text-dimmed">
      {{ subtitle }}
    </p>

    <!-- 登录/注册 切换(找回密码态隐藏) -->
    <div
      v-if="mode !== 'forgot'"
      class="relative grid grid-cols-2 p-1 mt-6 rounded-xl bg-elevated/60"
    >
      <span
        class="absolute top-1 bottom-1 left-1 w-[calc(50%-0.25rem)] rounded-lg bg-primary shadow-sm transition-transform duration-300 ease-out"
        :class="mode === 'register' ? 'translate-x-full' : 'translate-x-0'"
      />
      <button
        type="button"
        class="relative z-10 py-2 text-sm font-semibold rounded-lg transition-colors"
        :class="mode === 'login' ? 'text-inverted' : 'text-muted hover:text-highlighted'"
        @click="mode = 'login'"
      >
        登录
      </button>
      <button
        type="button"
        class="relative z-10 py-2 text-sm font-semibold rounded-lg transition-colors"
        :class="mode === 'register' ? 'text-inverted' : 'text-muted hover:text-highlighted'"
        @click="mode = 'register'"
      >
        注册
      </button>
    </div>

    <!-- 找回密码态:返回登录 -->
    <button
      v-else
      type="button"
      class="flex items-center gap-1 mt-6 text-sm font-medium text-muted hover:text-highlighted transition-colors"
      @click="mode = 'login'"
    >
      <UIcon
        name="i-lucide-arrow-left"
        class="w-4 h-4"
      />
      返回登录
    </button>

    <!-- 表单 -->
    <form
      class="flex flex-col mt-6 space-y-4"
      @submit.prevent="onSubmit"
    >
      <UFormField label="邮箱地址">
        <UInput
          v-model="email"
          type="email"
          size="lg"
          icon="i-lucide-mail"
          placeholder="请输入邮箱地址"
          :ui="{ root: 'w-full' }"
        />
      </UFormField>

      <!-- 验证码(注册 / 找回密码) -->
      <Transition
        enter-active-class="transition-all duration-300 ease-out overflow-hidden"
        leave-active-class="transition-all duration-200 ease-in overflow-hidden"
        enter-from-class="opacity-0 max-h-0"
        enter-to-class="opacity-100 max-h-24"
        leave-from-class="opacity-100 max-h-24"
        leave-to-class="opacity-0 max-h-0"
      >
        <UFormField
          v-if="needCode"
          label="验证码"
        >
          <div class="flex gap-2">
            <UInput
              v-model="code"
              inputmode="numeric"
              size="lg"
              icon="i-lucide-shield-check"
              placeholder="请输入 6 位验证码"
              :ui="{ root: 'flex-1' }"
            />
            <UButton
              size="lg"
              color="primary"
              variant="soft"
              class="shrink-0 font-medium whitespace-nowrap"
              :loading="sendingCode"
              :disabled="!canSendCode || sendingCode"
              @click="sendCode"
            >
              {{ countdown > 0 ? `${countdown}s 后重发` : '获取验证码' }}
            </UButton>
          </div>
        </UFormField>
      </Transition>

      <UFormField :label="passwordLabel">
        <UInput
          v-model="password"
          :type="showPassword ? 'text' : 'password'"
          size="lg"
          icon="i-lucide-lock"
          :placeholder="mode === 'forgot' ? '请输入新密码' : '请输入密码'"
          :ui="{ root: 'w-full' }"
        >
          <template #trailing>
            <UButton
              :icon="showPassword ? 'i-lucide-eye' : 'i-lucide-eye-off'"
              color="neutral"
              variant="link"
              size="sm"
              :aria-label="showPassword ? '隐藏密码' : '显示密码'"
              @click="showPassword = !showPassword"
            />
          </template>
        </UInput>
      </UFormField>

      <!-- 邀请码(仅注册) -->
      <Transition
        enter-active-class="transition-all duration-300 ease-out overflow-hidden"
        leave-active-class="transition-all duration-200 ease-in overflow-hidden"
        enter-from-class="opacity-0 max-h-0"
        enter-to-class="opacity-100 max-h-24"
        leave-from-class="opacity-100 max-h-24"
        leave-to-class="opacity-0 max-h-0"
      >
        <UFormField v-if="mode === 'register'">
          <template #label>
            邀请码
            <span class="text-xs font-normal text-dimmed">(选填)</span>
          </template>
          <UInput
            v-model="inviteCode"
            size="lg"
            icon="i-lucide-gift"
            placeholder="有邀请码可在此填写"
            :ui="{ root: 'w-full' }"
          />
        </UFormField>
      </Transition>

      <!-- 记住我 / 忘记密码(仅登录) -->
      <div
        v-if="mode === 'login'"
        class="flex items-center justify-between"
      >
        <UCheckbox
          v-model="remember"
          label="记住我"
        />
        <UButton
          color="primary"
          variant="link"
          size="sm"
          class="p-0"
          @click="mode = 'forgot'"
        >
          忘记密码?
        </UButton>
      </div>

      <UAlert
        v-if="message"
        color="success"
        variant="soft"
        icon="i-lucide-check-circle"
        :description="message"
      />

      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        icon="i-lucide-alert-circle"
        :description="error"
      />

      <!-- 提交按钮 -->
      <UButton
        type="submit"
        block
        size="lg"
        color="primary"
        class="font-semibold"
        :loading="submitting"
      >
        {{ submitLabel }}
      </UButton>
    </form>

    <!-- 其他方式登录(仅登录态) -->
    <template v-if="mode === 'login'">
      <div class="flex items-center gap-3 my-5">
        <USeparator class="flex-1" />
        <span class="text-xs text-dimmed whitespace-nowrap">或使用其他方式登录</span>
        <USeparator class="flex-1" />
      </div>

      <UButton
        block
        size="lg"
        color="neutral"
        variant="outline"
        class="font-medium"
      >
        <UIcon
          name="i-simple-icons-tencentqq"
          class="w-5 h-5"
        />
        QQ 登录
      </UButton>
    </template>

    <!-- 积分提示 -->
    <div class="mt-5 p-4 rounded-xl bg-elevated/60">
      <p class="text-sm font-semibold text-highlighted">
        首次注册:登录赠送 100 积分
      </p>
      <p class="mt-0.5 text-xs text-dimmed">
        免费生图,对新用户和会员用户限时免费!
      </p>
    </div>
  </div>
</template>
