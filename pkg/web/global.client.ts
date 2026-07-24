import htmx from 'htmx.org'

declare global {
  interface Window {
    htmx: typeof htmx
  }
}

interface HtmxRequestDetail {
  elt?: Element
  xhr?: XMLHttpRequest
}

interface HtmxBeforeSwapDetail {
  isError: boolean
  shouldSwap: boolean
  xhr: XMLHttpRequest
}

window.htmx = htmx

const requestDetail = (event: Event): HtmxRequestDetail | undefined =>
  (event as CustomEvent<HtmxRequestDetail>).detail

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
