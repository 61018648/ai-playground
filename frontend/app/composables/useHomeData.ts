// 首页所需的演示数据,集中管理便于后续替换为真实接口

export interface NavItem {
  label: string
  icon: string
  to: string
}

export interface BannerItem {
  id: number
  tags: string[]
  title: string
  pill: string
  /** Tailwind 渐变类,作为占位视觉 */
  gradient: string
}

export interface AppItem {
  id: number | string
  code: string
  name: string
  appType: 'image' | 'text'
  category: string
  description: string
  icon: string
  iconColor: string
}

export interface GalleryWork {
  id: number
  title: string
  category: string
  subtitle: string
  imageUrl: string
  samplePrompt: string
  targetAppCode: string
  accent: string
  featured?: boolean
}

export const useHomeData = () => {
  const navItems: NavItem[] = [
    { label: '画廊广场', icon: 'i-lucide-layout-grid', to: '/' },
    { label: '应用中心', icon: 'i-lucide-layout-dashboard', to: '/apps' },
    { label: '智能助手', icon: 'i-lucide-message-circle', to: '/assistant' },
    { label: '专业绘画', icon: 'i-lucide-palette', to: '/draw' },
    { label: '媒体库', icon: 'i-lucide-images', to: '/media' },
    { label: '更多工具', icon: 'i-lucide-wrench', to: '/tools' },
    { label: '生成历史', icon: 'i-lucide-clock', to: '/history' }
  ]

  const banners: BannerItem[] = [
    {
      id: 1,
      tags: ['商品广告', '商品图设计', '详情页', '宣传视频'],
      title: '电商视觉设计',
      pill: '商品视觉全链路AI化',
      gradient: 'from-amber-200 via-amber-100 to-orange-200'
    },
    {
      id: 2,
      tags: ['美食摄影', '美食海报', '营销视频', '食欲爆款'],
      title: 'Ai餐饮视觉',
      pill: '瞬息烹制食欲盛宴',
      gradient: 'from-orange-100 via-amber-50 to-amber-200'
    },
    {
      id: 3,
      tags: ['AI模特视频', '模特换装', '服装配饰', '穿搭带货'],
      title: 'Ai换装模特',
      pill: '服装电商爆款穿搭',
      gradient: 'from-rose-200 via-pink-100 to-rose-100'
    }
  ]

  const apps: AppItem[] = [
    {
      id: 1,
      code: 'creative-image',
      name: '创意图片生成',
      appType: 'image',
      category: '绘画作图',
      description: '用于海报、封面、插画和创意图...',
      icon: 'i-lucide-sparkles',
      iconColor: 'bg-emerald-100 text-emerald-600'
    },
    {
      id: 2,
      code: 'ai-drawing',
      name: 'AI 绘画作图',
      appType: 'image',
      category: '绘画作图',
      description: '用于海报、封面、插画和创意图...',
      icon: 'i-lucide-pen-tool',
      iconColor: 'bg-slate-900 text-white'
    },
    {
      id: 3,
      code: 'xiaohongshu-copy',
      name: '小红书爆款文案',
      appType: 'text',
      category: '文本创作',
      description: '作为一名专业的小红书爆款标题...',
      icon: 'i-lucide-book-heart',
      iconColor: 'bg-rose-100 text-rose-600'
    },
    {
      id: 4,
      code: 'high-eq-reply',
      name: '高情商回复',
      appType: 'text',
      category: '文本创作',
      description: '把对方的消息和对方是谁告诉我...',
      icon: 'i-lucide-messages-square',
      iconColor: 'bg-red-100 text-red-600'
    },
    {
      id: 5,
      code: 'wechat-title',
      name: '公众号标题生成器',
      appType: 'text',
      category: '文本创作',
      description: '作为一个资深运营,你可以告诉...',
      icon: 'i-lucide-newspaper',
      iconColor: 'bg-green-100 text-green-600'
    },
    {
      id: 6,
      code: 'writing-assistant',
      name: '全能写作助手',
      appType: 'text',
      category: '工作助手',
      description: '我可以为你提供写作灵感或文案...',
      icon: 'i-lucide-edit-3',
      iconColor: 'bg-yellow-100 text-yellow-600'
    }
  ]

  const styleTabs: string[] = ['全部', '电商营销', '海报设计', '摄影写真', '动漫游戏', '文本创作']

  const galleryWorks: GalleryWork[] = [
    {
      id: 1,
      title: '企业 AI 中转站海报',
      category: '海报设计',
      subtitle: '科技服务首屏主视觉',
      imageUrl: 'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=900&q=80',
      samplePrompt: '生成一张企业级 AI 中转站营销海报，亮色科技风，中心是云端数据枢纽，突出多模型接入、统一 API、稳定高速。',
      targetAppCode: 'ai-drawing',
      accent: 'bg-cyan-500',
      featured: true
    },
    {
      id: 2,
      title: '夏季饮品上新主图',
      category: '电商营销',
      subtitle: '适合外卖平台与门店海报',
      imageUrl: 'https://images.unsplash.com/photo-1544145945-f90425340c7e?auto=format&fit=crop&w=900&q=80',
      samplePrompt: '设计一张夏季柠檬气泡饮新品主图，明亮清爽，冰块、水珠、柠檬切片，适合电商和外卖平台使用。',
      targetAppCode: 'creative-image',
      accent: 'bg-emerald-500'
    },
    {
      id: 3,
      title: '服装模特棚拍',
      category: '摄影写真',
      subtitle: '商品细节与穿搭质感',
      imageUrl: 'https://images.unsplash.com/photo-1483985988355-763728e1935b?auto=format&fit=crop&w=900&q=80',
      samplePrompt: '生成一张高级女装电商棚拍图，年轻模特穿浅色通勤套装，柔和影棚光，干净背景，突出服装剪裁和质感。',
      targetAppCode: 'ai-drawing',
      accent: 'bg-rose-500'
    },
    {
      id: 4,
      title: '潮玩角色设定',
      category: '动漫游戏',
      subtitle: 'IP 形象与周边开发',
      imageUrl: 'https://images.unsplash.com/photo-1612036782180-6f0b6cd846fe?auto=format&fit=crop&w=900&q=80',
      samplePrompt: '设计一个潮玩 IP 角色，圆润可爱，街头潮流穿搭，带透明材质配件，适合盲盒产品设定图。',
      targetAppCode: 'ai-drawing',
      accent: 'bg-violet-500'
    },
    {
      id: 5,
      title: '小红书种草文案',
      category: '文本创作',
      subtitle: '标题、正文、标签一套生成',
      imageUrl: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=900&q=80',
      samplePrompt: '帮我写一篇小红书种草文案，主题是 AI 绘图工具，目标用户是电商运营和设计师，语气真实、有购买欲。',
      targetAppCode: 'xiaohongshu-copy',
      accent: 'bg-amber-500'
    },
    {
      id: 6,
      title: '公众号爆款标题',
      category: '文本创作',
      subtitle: '适合运营选题测试',
      imageUrl: 'https://images.unsplash.com/photo-1495020689067-958852a7765e?auto=format&fit=crop&w=900&q=80',
      samplePrompt: '围绕“企业如何用 AI 降低设计成本”生成 20 个公众号标题，兼顾专业感、点击率和可信度。',
      targetAppCode: 'wechat-title',
      accent: 'bg-blue-500'
    }
  ]

  return { navItems, banners, apps, styleTabs, galleryWorks }
}
