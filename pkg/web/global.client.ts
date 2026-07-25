import htmx from 'htmx.org'
import { requestDetail } from './lib/htmx-events.js'

declare global {
  interface Window {
    htmx: typeof htmx
  }
}



interface HtmxBeforeSwapDetail {
  isError: boolean
  shouldSwap: boolean
  xhr: XMLHttpRequest
}

window.htmx = htmx



const requestElement = (event: Event): Element | undefined =>
  requestDetail(event)?.elt

document.addEventListener('htmx:beforeSwap', (event) => {
  const detail = (event as CustomEvent<HtmxBeforeSwapDetail>).detail
  const contentType = detail?.xhr.getResponseHeader('Content-Type')
  if (detail?.xhr.status >= 400 && contentType?.startsWith('text/html')) {
    detail.shouldSwap = true
    detail.isError = false
  }
})

document.addEventListener('htmx:beforeRequest', (event) => {
  requestElement(event)?.setAttribute('aria-busy', 'true')
})



document.addEventListener('htmx:afterRequest', (event) => {
  const detail = requestDetail(event)
  const element = detail?.elt
  element?.removeAttribute('aria-busy')

  const status = detail?.xhr?.status
  if (
    element instanceof HTMLFormElement &&
    element.hasAttribute('data-reset-on-success') &&
    status !== undefined &&
    status >= 200 &&
    status < 300
  ) {
    element.reset()
  }
})

document.addEventListener('htmx:sendError', (event) => {
  requestElement(event)?.removeAttribute('aria-busy')
})
