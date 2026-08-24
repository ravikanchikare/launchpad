export function PageHeading({
  title,
  description,
  action,
  hideTitle,
}: {
  title: string
  description: string
  action?: React.ReactNode
  hideTitle?: boolean
}) {
  return (
    <div className="page-heading">
      <div className="page-heading-copy">
        {!hideTitle && <h1 className="page-title">{title}</h1>}
        <p className="page-description">{description}</p>
      </div>
      {action}
    </div>
  )
}
