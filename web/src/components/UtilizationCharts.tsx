import { useEffect, useState } from 'react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'maco-ui'
import { TimeSeriesChart } from 'maco-ui/charts'
import { errorMessage, getMetrics, getMetricInterfaces, type MetricPoint, type MetricRange, type MetricScope } from '../api'
import { formatBytes } from '../ux'

interface UtilizationChartsProps {
  scope: MetricScope
  target: string
}

interface RangeOption {
  value: MetricRange
  label: string
}

const ranges: RangeOption[] = [
  { value: '1h', label: 'Last hour' },
  { value: '6h', label: 'Last 6 hours' },
  { value: '24h', label: 'Last 24 hours' },
  { value: '7d', label: 'Last 7 days' },
]

function formatPercent(value: number) {
  return `${Number(value).toFixed(1)} %`
}

function formatThroughput(value: number) {
  return `${formatBytes(Number(value))}/s`
}

interface ChartTitleProps {
  title: string
  value?: string
}

function ChartTitle({ title, value }: ChartTitleProps) {
  return (
    <h3 className="flex items-baseline gap-2 text-sm font-semibold">
      {title}
      {value && <span className="font-normal tabular-nums text-muted-foreground">{value}</span>}
    </h3>
  )
}

function usePolled<T>(load: () => Promise<T>, initial: T, deps: unknown[]) {
  const [data, setData] = useState<T>(initial)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    let pending = false
    setLoading(true)
    async function refresh() {
      if (pending) return
      pending = true
      try {
        const next = await load()
        if (active) {
          setData(next)
          setError('')
        }
      } catch (e) {
        if (active) setError(errorMessage(e))
      } finally {
        pending = false
        if (active) setLoading(false)
      }
    }
    void refresh()
    const interval = window.setInterval(() => void refresh(), 5000)
    return () => {
      active = false
      window.clearInterval(interval)
    }
  }, deps)

  return { data, error, loading }
}

interface TrafficData {
  names: string[]
  selected: string
  points: MetricPoint[]
}

async function loadTraffic(preferred: string, range: MetricRange): Promise<TrafficData> {
  const names = await getMetricInterfaces()
  const selected = names.includes(preferred) ? preferred : names.includes('en0') ? 'en0' : (names[0] ?? '')
  const points = selected ? await getMetrics('interface', selected, range) : []
  return { names, selected, points }
}

function emptyText(loading: boolean, error: string, count: number): string {
  if (error) return error
  if (loading) return ''
  return count === 0 ? 'No samples in this period' : ''
}

export function UtilizationCharts({ scope, target }: UtilizationChartsProps) {
  const [range, setRange] = useState<MetricRange>('1h')
  const [iface, setIface] = useState('')
  const usage = usePolled(() => getMetrics(scope, target, range), [] as MetricPoint[], [scope, target, range])
  const traffic = usePolled(
    () => (scope === 'host' ? loadTraffic(iface, range) : Promise.resolve({ names: [], selected: '', points: [] })),
    { names: [], selected: '', points: [] } as TrafficData,
    [scope, range, iface],
  )
  const latest = usage.data[usage.data.length - 1]
  const latestTraffic = traffic.data.points[traffic.data.points.length - 1]
  const usageNote = emptyText(usage.loading, usage.error, usage.data.length)
  const trafficNote = emptyText(traffic.loading, traffic.error, traffic.data.points.length)

  return (
    <section className="utilization-charts space-y-5 rounded-xl bg-card p-4 [box-shadow:var(--card-shadow)]" aria-label="Utilization">
      <div className="flex items-center justify-end">
        <Select value={range} onValueChange={(value) => setRange(value as MetricRange)}>
          <SelectTrigger aria-label="Period" className="w-40"><SelectValue /></SelectTrigger>
          <SelectContent>
            {ranges.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
      {usageNote && <p role={usage.error ? 'alert' : 'status'} className={`text-sm ${usage.error ? 'text-danger' : 'text-muted-foreground'}`}>{usageNote}</p>}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="space-y-2">
          <ChartTitle title="CPU" value={latest?.cpu_percent != null ? formatPercent(latest.cpu_percent) : undefined} />
          <TimeSeriesChart data={usage.data} series={[{ dataKey: 'cpu_percent', label: 'CPU', color: 'var(--chart-primary)' }]} yFormatter={formatPercent} height={160} />
        </div>
        <div className="space-y-2">
          <ChartTitle title={scope === 'vm' ? 'Resident Memory' : 'Memory'} value={latest ? formatBytes(latest.memory_bytes) : undefined} />
          <TimeSeriesChart data={usage.data} series={[{ dataKey: 'memory_bytes', label: 'Memory', color: 'var(--chart-secondary)' }]} yFormatter={formatBytes} height={160} />
        </div>
      </div>
      {scope === 'host' && (
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <ChartTitle
              title="Network"
              value={
                latestTraffic && latestTraffic.rx_bytes_ps != null && latestTraffic.tx_bytes_ps != null
                  ? `↓ ${formatThroughput(latestTraffic.rx_bytes_ps)} ↑ ${formatThroughput(latestTraffic.tx_bytes_ps)}`
                  : undefined
              }
            />
            {traffic.data.names.length > 0 && (
              <Select value={traffic.data.selected} onValueChange={setIface}>
                <SelectTrigger aria-label="Interface" className="w-36 font-mono"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {traffic.data.names.map((name) => <SelectItem key={name} value={name} className="font-mono">{name}</SelectItem>)}
                </SelectContent>
              </Select>
            )}
          </div>
          {trafficNote && <p role={traffic.error ? 'alert' : 'status'} className={`text-sm ${traffic.error ? 'text-danger' : 'text-muted-foreground'}`}>{trafficNote}</p>}
          <TimeSeriesChart
            data={traffic.data.points}
            series={[
              { dataKey: 'rx_bytes_ps', label: 'Receive', color: 'var(--chart-primary)' },
              { dataKey: 'tx_bytes_ps', label: 'Transmit', color: 'var(--chart-secondary)' },
            ]}
            yFormatter={formatThroughput}
            height={180}
          />
        </div>
      )}
    </section>
  )
}
