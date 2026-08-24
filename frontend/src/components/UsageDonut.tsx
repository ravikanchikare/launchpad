import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Skeleton } from "@/components/ui/skeleton"
import { formatMoney } from "@/lib/format-money"
import type { AccountSummary } from "@/types"

const DONUT_SIZE = 24
const STROKE_WIDTH = 3

function formatBudgetReset(value?: string) {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

function UsageDonutRing({ percent }: { percent: number }) {
  const radius = (DONUT_SIZE - STROKE_WIDTH) / 2
  const circumference = 2 * Math.PI * radius
  const clamped = Math.min(100, Math.max(0, percent))
  const offset = circumference - (clamped / 100) * circumference

  return (
    <svg
      className="titlebar-usage-donut"
      width={DONUT_SIZE}
      height={DONUT_SIZE}
      viewBox={`0 0 ${DONUT_SIZE} ${DONUT_SIZE}`}
      aria-hidden="true"
    >
      <circle
        className="titlebar-usage-donut-track"
        cx={DONUT_SIZE / 2}
        cy={DONUT_SIZE / 2}
        r={radius}
        fill="none"
        strokeWidth={STROKE_WIDTH}
      />
      <circle
        className="titlebar-usage-donut-progress"
        cx={DONUT_SIZE / 2}
        cy={DONUT_SIZE / 2}
        r={radius}
        fill="none"
        strokeWidth={STROKE_WIDTH}
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        transform={`rotate(-90 ${DONUT_SIZE / 2} ${DONUT_SIZE / 2})`}
      />
    </svg>
  )
}

export function UsageDonut({ account }: { account: AccountSummary }) {
  const maxBudget = account.maxBudget
  if (!maxBudget || maxBudget <= 0) return null

  const percent = (account.spend / maxBudget) * 100
  const resetLabel = formatBudgetReset(account.budgetResetAt)
  const summary = `My usage: ${formatMoney(account.spend)} of ${formatMoney(maxBudget)}`

  return (
    <div className="titlebar-spend-meta">
      <Popover>
        <PopoverTrigger
          className="titlebar-usage-trigger"
          aria-label={summary}
        >
          <UsageDonutRing percent={percent} />
        </PopoverTrigger>
        <PopoverContent align="end" side="bottom" className="titlebar-usage-popover">
          <PopoverHeader>
            <PopoverTitle>My usage</PopoverTitle>
          </PopoverHeader>
          <dl className="titlebar-usage-details">
            <div className="titlebar-usage-detail">
              <dt>Spend</dt>
              <dd>{formatMoney(account.spend)}</dd>
            </div>
            <div className="titlebar-usage-detail">
              <dt>Budget</dt>
              <dd>{formatMoney(maxBudget)}</dd>
            </div>
            {resetLabel ? (
              <div className="titlebar-usage-detail">
                <dt>Resets</dt>
                <dd>{resetLabel}</dd>
              </div>
            ) : null}
          </dl>
        </PopoverContent>
      </Popover>
    </div>
  )
}

export function UsageDonutSkeleton() {
  return (
    <div className="titlebar-spend-meta" aria-hidden="true">
      <Skeleton className="size-6 rounded-full" />
    </div>
  )
}
