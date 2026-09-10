import { useState } from 'react'
import { ChevronLeft, ChevronRight, FileText, Trash2 } from 'lucide-react'
import { clsx } from 'clsx'
import type { ReceiptImage } from './types'

export function ImageCarousel({ images, onRemove, className }: { images: ReceiptImage[]; onRemove?: (id: string) => void; className?: string }) {
  const [i, setI] = useState(0)
  const idx = Math.min(i, Math.max(images.length - 1, 0))
  const img = images[idx]
  if (!img) {
    return <div className={clsx('grid aspect-[4/3] place-items-center rounded-lg bg-gray-100 text-sm text-gray-500 dark:bg-gray-800', className)}>No image</div>
  }
  const isPdf = img.contentType === 'application/pdf'
  return (
    <div className={clsx('relative', className)}>
      <div className="overflow-hidden rounded-lg bg-gray-100 dark:bg-gray-800">
        {isPdf ? (
          <div className="grid aspect-[4/3] place-items-center gap-2 text-sm">
            <iframe title={img.fileName ?? 'PDF'} src={img.url} className="h-full w-full min-h-[320px]" />
          </div>
        ) : (
          <a href={img.url} target="_blank" rel="noreferrer">
            <img src={img.url} alt={img.fileName ?? 'Receipt'} className="max-h-[520px] w-full object-contain" />
          </a>
        )}
      </div>
      {isPdf && (
        <a href={img.url} target="_blank" rel="noreferrer" className="mt-1 inline-flex items-center gap-1 text-xs text-brand-700 hover:underline">
          <FileText className="size-3.5" /> Open PDF
        </a>
      )}
      {images.length > 1 && (
        <>
          <button type="button" aria-label="Previous" onClick={() => setI((idx - 1 + images.length) % images.length)} className="absolute top-1/2 left-2 -translate-y-1/2 rounded-full bg-black/50 p-1 text-white">
            <ChevronLeft className="size-4" />
          </button>
          <button type="button" aria-label="Next" onClick={() => setI((idx + 1) % images.length)} className="absolute top-1/2 right-2 -translate-y-1/2 rounded-full bg-black/50 p-1 text-white">
            <ChevronRight className="size-4" />
          </button>
          <div className="mt-2 flex justify-center gap-1.5">
            {images.map((im, k) => (
              <button key={im.id} type="button" aria-label={`Image ${k + 1}`} onClick={() => setI(k)} className={clsx('size-2 rounded-full', k === idx ? 'bg-brand-600' : 'bg-gray-300 dark:bg-gray-600')} />
            ))}
          </div>
        </>
      )}
      {onRemove && (
        <button type="button" aria-label="Remove image" onClick={() => onRemove(img.id)} className="absolute top-2 right-2 rounded-full bg-black/50 p-1.5 text-white hover:bg-red-600">
          <Trash2 className="size-4" />
        </button>
      )}
    </div>
  )
}
