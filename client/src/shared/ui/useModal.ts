import { useEffect, useRef } from 'react'

// useModal opens and closes a native <dialog> as a modal to match `open`.
// The browser then keeps focus inside it and puts everything else behind it.
export function useModal(open: boolean) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])
  return ref
}
