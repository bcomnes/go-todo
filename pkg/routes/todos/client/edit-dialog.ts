import { requestDetail } from '../../../web/lib/htmx-events.js'

document.addEventListener('htmx:afterSwap', (event) => {
  const target = requestDetail(event)?.target
  if (target?.id !== 'todo-edit-content') return

  const dialog = document.querySelector<HTMLDialogElement>('#todo-edit-dialog')
  if (dialog && !dialog.open) dialog.showModal()
})

document.addEventListener('click', (event) => {
  const target = event.target
  if (!(target instanceof Element)) return

  const closeControl = target.closest('[data-dialog-close]')
  if (!closeControl) return

  const dialog = closeControl.closest<HTMLDialogElement>('dialog')
  if (!dialog) return

  event.preventDefault()
  dialog.close()
})

document.addEventListener('todoEditSaved', () => {
  document.querySelector<HTMLDialogElement>('#todo-edit-dialog')?.close()
})
