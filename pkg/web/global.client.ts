import htmx from 'htmx.org'

declare global {
  interface Window {
    htmx: typeof htmx
  }
}

interface HtmxRequestDetail {
  elt?: Element
}

interface HtmxBeforeSwapDetail {
  isError: boolean
  shouldSwap: boolean
  xhr: XMLHttpRequest
}

window.htmx = htmx

const requestElement = (event: Event): Element | undefined =>
  (event as CustomEvent<HtmxRequestDetail>).detail?.elt

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
  requestElement(event)?.removeAttribute('aria-busy')
})

document.addEventListener('htmx:sendError', (event) => {
  requestElement(event)?.removeAttribute('aria-busy')
})
