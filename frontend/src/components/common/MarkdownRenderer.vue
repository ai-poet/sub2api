<template>
  <div
    class="markdown-renderer"
    :class="className"
    v-html="html"
  ></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderSafeMarkdown } from '@/utils/markdown'

const props = defineProps<{
  content: string
  className?: string
}>()

const html = computed(() => renderSafeMarkdown(props.content || ''))
</script>

<style>
.markdown-renderer {
  color: inherit;
  font-size: inherit;
  line-height: 1.7;
}

.markdown-renderer > :first-child {
  margin-top: 0;
}

.markdown-renderer > :last-child {
  margin-bottom: 0;
}

.markdown-renderer h1 {
  margin: 1.2rem 0 0.65rem;
  font-size: 1.45rem;
  line-height: 1.25;
  font-weight: 800;
  color: #111827;
}

.markdown-renderer h2 {
  margin: 1rem 0 0.55rem;
  font-size: 1.25rem;
  line-height: 1.3;
  font-weight: 800;
  color: #111827;
}

.markdown-renderer h3 {
  margin: 0.9rem 0 0.45rem;
  font-size: 1.08rem;
  line-height: 1.35;
  font-weight: 700;
  color: #1f2937;
}

.markdown-renderer h4,
.markdown-renderer h5,
.markdown-renderer h6 {
  margin: 0.75rem 0 0.35rem;
  font-size: 0.98rem;
  line-height: 1.4;
  font-weight: 700;
  color: #1f2937;
}

.markdown-renderer p {
  margin: 0 0 0.7rem;
}

.markdown-renderer ul,
.markdown-renderer ol {
  margin: 0.45rem 0 0.8rem 1.25rem;
  padding-left: 1rem;
}

.markdown-renderer ul {
  list-style: disc;
}

.markdown-renderer ol {
  list-style: decimal;
}

.markdown-renderer li {
  margin: 0.25rem 0;
  padding-left: 0.15rem;
}

.markdown-renderer li::marker {
  color: #14b8a6;
}

.markdown-renderer a {
  color: #0d9488;
  font-weight: 600;
  text-decoration: underline;
  text-decoration-color: rgba(13, 148, 136, 0.35);
  text-underline-offset: 2px;
}

.markdown-renderer strong {
  font-weight: 700;
  color: #111827;
}

.markdown-renderer em {
  font-style: italic;
}

.markdown-renderer code {
  border-radius: 0.35rem;
  background: rgba(15, 23, 42, 0.06);
  padding: 0.08rem 0.32rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.9em;
  color: #be123c;
}

.markdown-renderer pre {
  margin: 0.85rem 0;
  overflow-x: auto;
  border-radius: 0.65rem;
  border: 1px solid #e5e7eb;
  background: #f8fafc;
  padding: 0.85rem;
}

.markdown-renderer pre code {
  background: transparent;
  padding: 0;
  color: #334155;
}

.markdown-renderer blockquote {
  margin: 0.85rem 0;
  border-left: 3px solid #14b8a6;
  background: rgba(20, 184, 166, 0.08);
  padding: 0.55rem 0.8rem;
  color: #475569;
}

.markdown-renderer hr {
  margin: 1rem 0;
  border: 0;
  border-top: 1px solid #e5e7eb;
}

.markdown-renderer table {
  margin: 0.85rem 0;
  width: 100%;
  border-collapse: collapse;
  font-size: 0.95em;
}

.markdown-renderer th,
.markdown-renderer td {
  border: 1px solid #e5e7eb;
  padding: 0.45rem 0.6rem;
  text-align: left;
}

.markdown-renderer th {
  background: #f8fafc;
  font-weight: 700;
  color: #111827;
}

.markdown-renderer img {
  margin: 0.85rem 0;
  max-width: 100%;
  border-radius: 0.65rem;
}

.dark .markdown-renderer h1,
.dark .markdown-renderer h2 {
  color: #f8fafc;
}

.dark .markdown-renderer h3,
.dark .markdown-renderer h4,
.dark .markdown-renderer h5,
.dark .markdown-renderer h6,
.dark .markdown-renderer strong {
  color: #f1f5f9;
}

.dark .markdown-renderer a {
  color: #5eead4;
  text-decoration-color: rgba(94, 234, 212, 0.35);
}

.dark .markdown-renderer li::marker {
  color: #2dd4bf;
}

.dark .markdown-renderer code {
  background: rgba(255, 255, 255, 0.08);
  color: #fda4af;
}

.dark .markdown-renderer pre {
  border-color: #334155;
  background: rgba(15, 23, 42, 0.7);
}

.dark .markdown-renderer pre code {
  color: #e2e8f0;
}

.dark .markdown-renderer blockquote {
  background: rgba(20, 184, 166, 0.12);
  color: #cbd5e1;
}

.dark .markdown-renderer hr,
.dark .markdown-renderer th,
.dark .markdown-renderer td {
  border-color: #334155;
}

.dark .markdown-renderer th {
  background: rgba(15, 23, 42, 0.75);
  color: #f8fafc;
}
</style>
