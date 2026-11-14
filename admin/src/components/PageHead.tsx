import { Fragment } from 'react'
import { BackButton } from './BackButton'

interface PageHeadProps {
  crumbs?: React.ReactNode[]
  title: React.ReactNode
  sub?: React.ReactNode
  /** Logical parent route for the "‹ Back" button (where it goes when there is
   *  no in-app history to return to). Omit only on the Dashboard landing page. */
  back?: string
  children?: React.ReactNode
}

export function PageHead({ crumbs, title, sub, back, children }: PageHeadProps) {
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
      {(back || children) && (
        <div className="ph-r">
          {back && <BackButton fallback={back} />}
          {children}
        </div>
      )}
    </div>
  )
}
