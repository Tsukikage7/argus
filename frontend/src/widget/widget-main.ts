import { defineCustomElement } from 'vue'
import ArgusWidget from './ArgusWidget.ce.vue'

// 注册自定义元素
const ArgusWidgetElement = defineCustomElement(ArgusWidget)
customElements.define('argus-widget', ArgusWidgetElement)

// 读取当前 script 标签属性并自动挂载
const currentScript = document.currentScript as HTMLScriptElement | null
if (currentScript) {
  const apiKey = currentScript.getAttribute('data-api-key') ?? ''
  const baseUrl = currentScript.getAttribute('data-base-url') ?? '/api/v1'

  const el = document.createElement('argus-widget')
  el.setAttribute('api-key', apiKey)
  el.setAttribute('base-url', baseUrl)

  // 插入到 script 标签之后
  currentScript.insertAdjacentElement('afterend', el)
}

export { ArgusWidgetElement }
