// Canvas Studio translation keys.
export default {
  canvas: {
    nav: 'Canvas Studio',
    title: 'Canvas Studio',
    subtitle: 'Compose layers on a canvas, paint the region to repaint, then submit.',
    tools: {
      select: 'Select',
      brush: 'Mask brush',
      eraser: 'Eraser',
      text: 'Text label',
    },
    actions: {
      importImage: 'Import image',
      importHint: 'PNG / JPEG / WebP, up to 50MB per file',
      undo: 'Undo',
      redo: 'Redo',
      delete: 'Delete selected layer',
      clear: 'Clear canvas',
    },
    empty: {
      title: 'The canvas is empty',
      hint: 'Import an image as the base, then paint the region to repaint.',
    },
    panel: {
      apiKey: 'API Key',
      apiKeyPlaceholder: 'Pick a key that allows image generation',
      model: 'Model',
      modelPlaceholder: 'Pick an image model',
      prompt: 'Prompt',
      promptPlaceholder: 'Describe the result you want……',
      size: 'Size',
      quality: 'Quality',
      count: 'Images',
      submit: 'Generate',
      submitting: 'Submitting……',
    },
    quality: {
      default: 'Default',
      low: 'Low',
      medium: 'Medium',
      high: 'High',
    },
    status: {
      processing: 'Generating… results will appear below.',
    },
    history: {
      title: 'This session',
      useAsSource: 'Send back to the canvas',
    },
    textPrompt: 'Text label to add:',
    errors: {
      noResults: 'The task finished, but no usable image could be retrieved.',
      importFailed: 'Could not read the image. Try a PNG, JPEG, or WebP file.',
      fileTooLarge: 'The image is over 50MB. Compress it and try again.',
      noApiKey: 'Pick an API key that allows image generation.',
      compositeFailed: 'Could not compose the canvas. Please try again.',
      taskFailed: 'The generation task failed.',
    },
  },
}
