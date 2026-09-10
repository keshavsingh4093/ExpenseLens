export function Placeholder({ title }: { title: string }) {
  return (
    <div className="card p-6">
      <h1 className="text-xl font-semibold">{title}</h1>
      <p className="text-muted mt-1 text-sm">This screen is not built yet.</p>
    </div>
  )
}
