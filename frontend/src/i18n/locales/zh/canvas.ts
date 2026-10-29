// 画布工作台（Canvas Studio）的翻译键。
// 独立成文件：与 batchImage 同样挂在 locale 根级，深合并进主语言包。
export default {
  canvas: {
    nav: '画布工作台',
    title: '画布工作台',
    subtitle: '在画布上组合图层、涂抹要重绘的区域，然后提交生成。',
    tools: {
      select: '选择',
      brush: '遮罩笔刷',
      eraser: '擦除',
      text: '文本标注',
    },
    actions: {
      importImage: '导入图片',
      importHint: '支持 PNG / JPEG / WebP，单张不超过 50MB',
      undo: '撤销',
      redo: '重做',
      delete: '删除选中图层',
      clear: '清空画布',
    },
    empty: {
      title: '画布还是空的',
      hint: '导入一张图片作为底图，然后用笔刷涂抹要重绘的区域。',
    },
    panel: {
      apiKey: 'API Key',
      apiKeyPlaceholder: '选择一个可生图的 Key',
      model: '模型',
      modelPlaceholder: '选择一个生图模型',
      prompt: '提示词',
      promptPlaceholder: '描述你想要的画面效果……',
      size: '尺寸',
      quality: '质量',
      count: '生成张数',
      submit: '开始生成',
      submitting: '提交中……',
    },
    quality: {
      default: '默认',
      low: '低',
      medium: '中',
      high: '高',
    },
    status: {
      processing: '任务生成中，结果会自动出现在下方。',
    },
    history: {
      title: '本次生成',
      useAsSource: '放回画布继续编辑',
    },
    textPrompt: '输入要添加的文本标注：',
    errors: {
      noResults: '任务已完成，但没有取回可用图片。',
      importFailed: '图片读取失败，请换一张 PNG / JPEG / WebP 图片。',
      fileTooLarge: '图片超过 50MB，请压缩后再导入。',
      noApiKey: '请选择一个允许生图的 API Key。',
      compositeFailed: '画布合成失败，请重试。',
      taskFailed: '生成任务失败。',
    },
  },
}
