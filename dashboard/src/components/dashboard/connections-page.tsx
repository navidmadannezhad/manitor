import { useEffect, useMemo, useState } from 'react'
import { IoEyeOutline } from 'react-icons/io5'

import { ConnectionTrafficModal } from '@/components/dashboard/connection-traffic-modal'
import { Button } from '@/components/ui/button'

type Connection = {
  id?: string | number
  ip?: string
  wifiName?: string
  wifi_name?: string
  upload_size?: number | string
  download_size?: number | string
  total_download?: number | string
  total_upload?: number | string
  totalUploadSize?: number | string
  total_upload_size?: number | string
}

const serverBaseUrl = import.meta.env.VITE_SERVER_BASE_URL

function toNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return 0
}

function formatBytes(input: unknown): string {
  const bytes = toNumber(input)
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exp = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / 1024 ** exp
  return `${value >= 10 ? value.toFixed(0) : value.toFixed(1)} ${units[exp]}`
}

export function ConnectionsPage() {
  const [rows, setRows] = useState<Connection[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [chartOpen, setChartOpen] = useState(false)
  const [chartIp, setChartIp] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    async function fetchConnections() {
      setLoading(true)
      setError(null)
      try {
        if (!serverBaseUrl) {
          throw new Error('VITE_SERVER_BASE_URL is not configured')
        }

        const response = await fetch(
          `${serverBaseUrl}/api/v1/connections`,
          { signal: controller.signal }
        )

        if (!response.ok) {
          throw new Error(`Request failed (${response.status})`)
        }

        const payload = await response.json()
        const data = Array.isArray(payload)
          ? payload
          : Array.isArray(payload?.data)
            ? payload.data
            : []
        setRows(data)
      } catch (err) {
        if ((err as Error).name === 'AbortError') return
        setError((err as Error).message || 'Failed to fetch connections')
      } finally {
        setLoading(false)
      }
    }

    fetchConnections()

    return () => controller.abort()
  }, [])

  const content = useMemo(() => {
    if (loading) {
      return (
        <tr>
          <td colSpan={5} className="px-4 py-10 text-center text-sm text-muted-foreground">
            Loading connections...
          </td>
        </tr>
      )
    }

    if (error) {
      return (
        <tr>
          <td colSpan={5} className="px-4 py-10 text-center text-sm text-destructive">
            {error}
          </td>
        </tr>
      )
    }

    if (!rows.length) {
      return (
        <tr>
          <td colSpan={5} className="px-4 py-10 text-center text-sm text-muted-foreground">
            No connection records found.
          </td>
        </tr>
      )
    }

    return rows.map((row, index) => (
      <tr key={String(row.id ?? `${row.ip}-${index}`)} className="border-t border-border/70">
        <td className="px-4 py-3 text-sm text-foreground">{row.ip ?? '-'}</td>
        <td className="px-4 py-3 text-sm text-foreground">
          {row.wifiName ?? row.wifi_name ?? '-'}
        </td>
        <td className="px-4 py-3 text-sm text-foreground">
          {formatBytes(row.total_download ?? row.download_size ?? '-')}
        </td>
        <td className="px-4 py-3 text-sm text-foreground">
          {formatBytes(row.total_upload ?? row.upload_size ?? '-')}
        </td>
        <td className="px-4 py-3 text-right">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="View live traffic"
            disabled={!serverBaseUrl || !row.ip}
            onClick={() => {
              if (!row.ip) return
              setChartIp(String(row.ip))
              setChartOpen(true)
            }}
          >
            <IoEyeOutline className="size-4 text-muted-foreground" />
          </Button>
        </td>
      </tr>
    ))
  }, [error, loading, rows])

  return (
    <section id="connections" className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      {serverBaseUrl && (
        <ConnectionTrafficModal
          open={chartOpen}
          onOpenChange={(open) => {
            setChartOpen(open)
            if (!open) setChartIp(null)
          }}
          ip={chartIp}
          baseUrl={serverBaseUrl}
        />
      )}
      <div className="rounded-lg border border-border bg-card/40 p-4 sm:p-6">
        <h2 className="text-lg font-semibold text-foreground">Connections</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Monitor known network connections and bandwidth consumption.
        </p>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-card/30">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] border-collapse">
            <thead className="bg-secondary/50">
              <tr className="text-left">
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  IP
                </th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Wifi Name
                </th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Total Download Size
                </th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Total Upload Size
                </th>
                <th className="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Operation
                </th>
              </tr>
            </thead>
            <tbody>{content}</tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
