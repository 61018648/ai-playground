// 移动端侧边栏开关的共享状态(useState 保证 SSR 安全 & 跨组件共享)
export const useSidebar = () => {
  const isOpen = useState('sidebar-open', () => false)

  const open = () => {
    isOpen.value = true
  }
  const close = () => {
    isOpen.value = false
  }
  const toggle = () => {
    isOpen.value = !isOpen.value
  }

  return { isOpen, open, close, toggle }
}
