export interface HtmxRequestDetail {
  elt?: Element
  target?: Element
  xhr?: XMLHttpRequest
}

export const requestDetail = (event: Event): HtmxRequestDetail | undefined =>
  (event as CustomEvent<HtmxRequestDetail>).detail
