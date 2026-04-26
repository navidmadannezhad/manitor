import { useCallback, useEffect, useState } from 'react'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
  type ChartData,
} from 'chart.js'
import { Line } from 'react-chartjs-2'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const CHART_MAX_POINTS = 200

type ServerRow = {
  id: number
  created_at: string
  download_size: number
  upload_size: number
}

type WsMessage =
  | { type: 'history'; ip: string; data: ServerRow[] }
  | { type: 'update'; ip: string; data: ServerRow[] }
  | { type: 'error'; message: string }

type ChartPoint = { id: number; t: string; up: number; down: number }

function httpToWebSocketBase(url: string): string {
  const t = url.trim()
  if (t.startsWith('https://')) return 'wss://' + t.slice('https://'.length)
  if (t.startsWith('http://')) return 'ws://' + t.slice('http://'.length)
  return t
}

function toNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const n = Number(value)
    return Number.isFinite(n) ? n : 0
  }
  return 0
}

function formatBytesShort(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exp = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const v = bytes / 1024 ** exp
  return `${v >= 10 ? v.toFixed(0) : v.toFixed(1)} ${units[exp]}`
}

function labelForRow(createdAt: string, index: number): string {
  const d = new Date(createdAt)
  if (!Number.isNaN(d.getTime())) {
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  }
  return String(index)
}

function rowsToPoints(rows: ServerRow[]): ChartPoint[] {
  return rows.map((r, i) => ({
    id: toNumber(r.id),
    t: labelForRow(r.created_at, i),
    up: toNumber(r.upload_size),
    down: toNumber(r.download_size),
  }))
}

type ConnectionTrafficModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  ip: string | null
  baseUrl: string
}

export function ConnectionTrafficModal({ open, onOpenChange, ip, baseUrl }: ConnectionTrafficModalProps) {
  const [points, setPoints] = useState<ChartPoint[]>([])
  const [error, setError] = useState<string | null>(null)
  const [awaitingFirst, setAwaitingFirst] = useState(true)

  const reset = useCallback(() => {
    setPoints([])
    setError(null)
    setAwaitingFirst(true)
  }, [])

  useEffect(() => {
    if (!open) return
    reset()
  }, [open, ip, reset])

  useEffect(() => {
    if (!open || !ip?.trim() || !baseUrl.trim()) return

    const wsBase = httpToWebSocketBase(baseUrl.replace(/\/$/, ''))
    const path = `${wsBase}/api/v1/connections/${encodeURIComponent(ip.trim())}`
    const ws = new WebSocket(path)

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string) as WsMessage
        if (msg.type === 'error') {
          setError(msg.message || 'WebSocket error')
          setAwaitingFirst(false)
          return
        }
        if (msg.type === 'history') {
          const list = Array.isArray(msg.data) ? msg.data : []
          setPoints(rowsToPoints(list).slice(-CHART_MAX_POINTS))
          setAwaitingFirst(false)
          return
        }
        if (msg.type === 'update' && Array.isArray(msg.data) && msg.data.length > 0) {
          setPoints((prev) => {
            const next = [...prev, ...rowsToPoints(msg.data)]
            return next.slice(-CHART_MAX_POINTS)
          })
          setAwaitingFirst(false)
        }
      } catch {
        setError('Could not read server data')
        setAwaitingFirst(false)
      }
    }

    ws.onerror = () => {
      setError('WebSocket connection failed')
      setAwaitingFirst(false)
    }

    return () => {
      ws.close()
    }
  }, [open, ip, baseUrl])

  const data: ChartData<'line', number[], string> = {
    labels: points.map((p) => p.t),
    datasets: [
      {
        label: 'Upload',
        data: points.map((p) => p.up),
        borderColor: 'rgb(37, 99, 235)',
        backgroundColor: 'rgba(37, 99, 235, 0.1)',
        fill: true,
        tension: 0.2,
        pointRadius: 0,
        borderWidth: 2,
      },
      {
        label: 'Download',
        data: points.map((p) => p.down),
        borderColor: 'rgb(220, 38, 38)',
        backgroundColor: 'rgba(220, 38, 38, 0.08)',
        fill: true,
        tension: 0.2,
        pointRadius: 0,
        borderWidth: 2,
      },
    ],
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Connection traffic{ip ? ` — ${ip}` : ''}</DialogTitle>
          <DialogDescription>
            Per-interval upload and download (last {CHART_MAX_POINTS} samples), updated every second from the server.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <p className="text-sm text-destructive" role="alert">
            {error}
          </p>
        )}

        {awaitingFirst && !error && (
          <p className="text-sm text-muted-foreground">Connecting…</p>
        )}

        {!awaitingFirst && !error && points.length === 0 && (
          <p className="text-sm text-muted-foreground">No traffic samples for this user yet.</p>
        )}

        {points.length > 0 && (
          <div className="h-[min(40vh,360px)] w-full min-h-[220px]">
            <Line
              data={data}
              options={{
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                  legend: {
                    position: 'top',
                    labels: { usePointStyle: true, boxWidth: 8 },
                  },
                  tooltip: {
                    callbacks: {
                      label: (ctx) => {
                        const y = toNumber(ctx.parsed.y)
                        return `${ctx.dataset.label ?? ''}: ${formatBytesShort(y)}`
                      },
                    },
                  },
                },
                scales: {
                  x: {
                    grid: { color: 'rgba(148, 163, 184, 0.2)' },
                    ticks: { maxRotation: 45, minRotation: 0, maxTicksLimit: 10 },
                  },
                  y: {
                    beginAtZero: true,
                    grid: { color: 'rgba(148, 163, 184, 0.2)' },
                    ticks: {
                      callback: (v) => (typeof v === 'number' ? formatBytesShort(v) : String(v)),
                    },
                  },
                },
              }}
            />
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
