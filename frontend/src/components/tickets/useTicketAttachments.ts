/**
 * 工单图片附件（fork 本地功能）：线程回复框与新建工单表单共用的上传逻辑。
 *
 * 上传期间在草稿里插入唯一占位符 `![图片上传中…](uploading-N)`，成功后替换为
 * `![image](ticket-attachment://<key>)`，失败则移除占位并 toast 本地化错误。
 * 渲染侧的 ticket-attachment:// 改写见 utils/tickets.ts 的 rewriteTicketAttachmentUrls。
 */

import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { uploadAttachment as uploadUserAttachment } from '@/api/tickets'
import { uploadAttachment as uploadAdminAttachment } from '@/api/admin/tickets'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  TICKET_ATTACHMENT_MAX_SIZE,
  ticketAttachmentMarkdown,
  type TicketSide
} from '@/utils/tickets'

export interface UseTicketAttachmentsOptions {
  side: TicketSide
  /** 要插入 Markdown 的草稿模型（线程回复框 / 新建工单 body） */
  draft: Ref<string>
}

export function useTicketAttachments({ side, draft }: UseTicketAttachmentsOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const fileInput = ref<HTMLInputElement | null>(null)
  /** 进行中的上传数量：>0 时禁止发送，避免把占位符发出去 */
  const uploadingCount = ref(0)
  let placeholderSeq = 0

  const upload = side === 'admin' ? uploadAdminAttachment : uploadUserAttachment

  function pickImage() {
    fileInput.value?.click()
  }

  function appendToDraft(snippet: string) {
    const current = draft.value
    const separator = current.length > 0 && !current.endsWith('\n') ? '\n' : ''
    draft.value = `${current}${separator}${snippet}`
  }

  function errorMessage(err: unknown): string {
    return extractApiErrorMessage(err, t('tickets.attachments.uploadFailed'), {
      TICKET_ATTACHMENT_STORAGE_NOT_CONFIGURED: t('tickets.attachments.errors.storageNotConfigured'),
      TICKET_ATTACHMENT_TOO_LARGE: t('tickets.attachments.errors.tooLarge'),
      TICKET_ATTACHMENT_BAD_TYPE: t('tickets.attachments.errors.badType')
    })
  }

  async function uploadFiles(fileList: FileList | File[] | null | undefined) {
    for (const file of Array.from(fileList ?? [])) {
      if (!file.type.startsWith('image/')) {
        appStore.showError(t('tickets.attachments.errors.badType'))
        continue
      }
      if (file.size > TICKET_ATTACHMENT_MAX_SIZE) {
        appStore.showError(t('tickets.attachments.errors.tooLarge'))
        continue
      }
      placeholderSeq += 1
      const placeholder = `![${t('tickets.attachments.uploading')}](uploading-${placeholderSeq})`
      appendToDraft(placeholder)
      uploadingCount.value += 1
      try {
        const result = await upload(file)
        draft.value = draft.value.replace(placeholder, ticketAttachmentMarkdown(result.key))
      } catch (err) {
        draft.value = draft.value.replace(placeholder, '')
        appStore.showError(errorMessage(err))
      } finally {
        uploadingCount.value -= 1
      }
    }
  }

  function imageFiles(fileList: FileList | File[] | null | undefined): File[] {
    // 粘贴 / 拖拽混入的非图片静默忽略（文件选择器走 accept="image/*"，走错类型会toast）
    return Array.from(fileList ?? []).filter((file) => file.type.startsWith('image/'))
  }

  async function onFileChange(event: Event) {
    const input = event.target as HTMLInputElement
    await uploadFiles(input.files)
    input.value = ''
  }

  async function onPaste(event: ClipboardEvent) {
    const files = imageFiles(event.clipboardData?.files)
    if (files.length === 0) return
    event.preventDefault()
    await uploadFiles(files)
  }

  async function onDrop(event: DragEvent) {
    const files = imageFiles(event.dataTransfer?.files)
    if (files.length === 0) return
    event.preventDefault()
    await uploadFiles(files)
  }

  return { fileInput, uploadingCount, pickImage, onFileChange, onPaste, onDrop }
}
