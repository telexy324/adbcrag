import { InputHTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn('h-9 w-full rounded-md border border-border bg-white px-3 text-sm outline-none focus:ring-2 focus:ring-teal-600', className)} {...props} />
}
