// /draw 页面分类卡片数据
export interface DrawCategory {
  id: number
  name: string
  description: string
  icon: string
  /** 图标底色 */
  color: string
  /** 选中后写入输入框的预设提示词 */
  prompt: string
}

export const useDrawData = () => {
  const categories: DrawCategory[] = [
    {
      id: 1,
      name: '电商设计',
      description: '卖货素材一键做',
      icon: 'i-lucide-shopping-bag',
      color: 'bg-red-400',
      prompt: '帮我设计一张电商商品主图,突出产品卖点,风格简洁高级,适合电商平台展示。'
    },
    {
      id: 2,
      name: '海报设计',
      description: '海报一键速出',
      icon: 'i-lucide-image',
      color: 'bg-cyan-400',
      prompt: '帮我设计一张活动宣传海报,主题鲜明、视觉冲击力强,包含标题和主视觉。'
    },
    {
      id: 3,
      name: '社媒营销',
      description: '图文产出全包',
      icon: 'i-lucide-heart',
      color: 'bg-blue-500',
      prompt: '帮我生成一套社交媒体营销图文,风格年轻时尚,适合小红书/微博发布。'
    },
    {
      id: 4,
      name: '品牌设计',
      description: '从零打造品牌',
      icon: 'i-lucide-flag',
      color: 'bg-green-500',
      prompt: '帮我设计一套品牌视觉方案,包含 Logo 概念、主色调和字体风格建议。'
    },
    {
      id: 5,
      name: '办公设计',
      description: '学生职场通用',
      icon: 'i-lucide-pie-chart',
      color: 'bg-blue-400',
      prompt: '帮我设计一份专业的演示文稿封面,排版清晰,适合职场或学生汇报使用。'
    }
  ]

  return { categories }
}
