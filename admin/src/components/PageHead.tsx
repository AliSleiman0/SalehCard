import { Fragment } from 'react'

interface PageHeadProps {
  crumbs?: React.ReactNode[]
  title: React.ReactNode
  sub?: React.ReactNode
  children?: React.ReactNode
}

export function PageHead({ crumbs, title, sub, children }: PageHeadProps) {
  return (
    <div className="pagehead">
      <div className="ph-l">
        {crumbs && (
          <div className="crumb">
            {crumbs.map((c, i) => (
              <Fragment key={i}>
                {i > 0 && <span className="sep">/</span>}
                <span>{c}</span>
              </Fragment>
            ))}
          </div>
        )}
        <h1 className="page-title">{title}</h1>
        {sub && <div className="page-sub">{sub}</div>}
      </div>
      {children && <div className="ph-r">{children}</div>}
    </div>
  )
}
