import { ref, onMounted, onUnmounted } from 'vue'

export type ProjectColorStyle = 'gradient' | 'solid' | 'off'
export type ProjectColorGlow = 'on' | 'off'

const colorStyle = ref<ProjectColorStyle>(
  (typeof localStorage !== 'undefined' ? localStorage.getItem('cct_project_color_style') : null) as ProjectColorStyle || 'off'
)
const colorGlow = ref<ProjectColorGlow>(
  (typeof localStorage !== 'undefined' ? localStorage.getItem('cct_project_color_glow') : null) as ProjectColorGlow || 'off'
)

function refresh() {
  colorStyle.value = (localStorage.getItem('cct_project_color_style') as ProjectColorStyle) || 'off'
  colorGlow.value = (localStorage.getItem('cct_project_color_glow') as ProjectColorGlow) || 'off'
}

export function useProjectColorSettings() {
  const handler = () => refresh()

  onMounted(() => {
    window.addEventListener('cct-project-color-settings-changed', handler)
    refresh()
  })

  onUnmounted(() => {
    window.removeEventListener('cct-project-color-settings-changed', handler)
  })

  return {
    colorStyle,
    colorGlow,
  }
}
