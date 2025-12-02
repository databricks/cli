import { useAnalyticsQuery, AreaChart, LineChart, RadarChart } from '@databricks/app-kit-ui/react';
import { Line, XAxis, YAxis, CartesianGrid, Tooltip } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { trpc } from './lib/trpc';
import { useState, useEffect } from 'react';

function App() {
  const { data, loading, error } = useAnalyticsQuery('hello_world', {
    message: 'hello world',
  });

  const [health, setHealth] = useState<{
    status: string;
    timestamp: string;
  } | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);

  useEffect(() => {
    trpc.healthcheck
      .query()
      .then(setHealth)
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setHealthError(message);
      });
  }, []);

  const [modelResponse, setModelResponse] = useState<string | null>(null);
  const [modelLoading, setModelLoading] = useState<boolean>(true);
  const [modelError, setModelError] = useState<string | null>(null);

  useEffect(() => {
    trpc.queryModel
      .query({ prompt: 'How are you today?' })
      .then((response: string) => {
        setModelResponse(response);
        setModelLoading(false);
      })
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setModelError(message);
        setModelLoading(false);
      });
  }, []);

  const [maxMonthNum, setMaxMonthNum] = useState<number>(12);

  const salesParameters = { max_month_num: maxMonthNum };

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4 w-full">
      <div className="mb-8 text-center">
        <h1 className="text-4xl font-bold mb-2 text-foreground">Minimal Databricks App</h1>
        <p className="text-lg text-muted-foreground max-w-md">A minimal Databricks App powered by Databricks AppKit</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 w-full max-w-7xl">
        <Card className="shadow-lg">
          <CardHeader>
            <CardTitle>SQL Query Result</CardTitle>
          </CardHeader>
          <CardContent>
            {loading && (
              <div className="space-y-2">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-8 w-1/2" />
              </div>
            )}
            {error && <div className="text-destructive bg-destructive/10 p-3 rounded-md">Error: {error}</div>}
            {data && data.length > 0 && (
              <div className="space-y-2">
                <div className="text-sm text-muted-foreground">Query: SELECT :message AS value</div>
                <div className="text-2xl font-bold text-primary">{data[0].value}</div>
              </div>
            )}
            {data && data.length === 0 && <div className="text-muted-foreground">No results</div>}
          </CardContent>
        </Card>

        <Card className="shadow-lg">
          <CardHeader>
            <CardTitle>Health Check</CardTitle>
          </CardHeader>
          <CardContent>
            {!health && !healthError && (
              <div className="space-y-2">
                <Skeleton className="h-6 w-24" />
                <Skeleton className="h-4 w-48" />
              </div>
            )}
            {healthError && (
              <div className="text-destructive bg-destructive/10 p-3 rounded-md">Error: {healthError}</div>
            )}
            {health && (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full bg-[hsl(var(--success))] animate-pulse"></div>
                  <div className="text-lg font-semibold text-[hsl(var(--success))]">{health.status.toUpperCase()}</div>
                </div>
                <div className="text-sm text-muted-foreground">
                  Last checked: {new Date(health.timestamp).toLocaleString()}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="shadow-lg">
          <CardHeader>
            <CardTitle>Model Query Demo</CardTitle>
          </CardHeader>
          <CardContent>
            {modelLoading && (
              <div className="space-y-2">
                <Skeleton className="h-4 w-48" />
                <div className="bg-muted p-3 rounded-md border border-border space-y-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-5/6" />
                  <Skeleton className="h-4 w-4/6" />
                </div>
              </div>
            )}
            {modelError && <div className="text-destructive bg-destructive/10 p-3 rounded-md">Error: {modelError}</div>}
            {modelResponse && (
              <div className="space-y-2">
                <div className="text-sm text-muted-foreground">Prompt: &quot;How are you today?&quot;</div>
                <div className="text-base bg-muted p-3 rounded-md border border-border">{modelResponse}</div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="shadow-lg md:col-span-3">
          <CardHeader>
            <CardTitle>Sales Data Filter</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="max-w-md">
              <div className="space-y-2">
                <Label htmlFor="max-month">Show data up to month</Label>
                <Select value={maxMonthNum.toString()} onValueChange={(value) => setMaxMonthNum(parseInt(value))}>
                  <SelectTrigger id="max-month">
                    <SelectValue placeholder="All months" />
                  </SelectTrigger>
                  <SelectContent>
                    {[...Array(12)].map((_, i) => (
                      <SelectItem key={i + 1} value={(i + 1).toString()}>
                        {i + 1 === 12 ? 'All months (12)' : `Month ${i + 1}`}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="shadow-lg flex min-w-0">
          <CardHeader>
            <CardTitle>Sales Trend Area Chart</CardTitle>
          </CardHeader>
          <CardContent>
            <AreaChart queryKey="mocked_sales" parameters={salesParameters} />
          </CardContent>
        </Card>
        <Card className="shadow-lg flex min-w-0">
          <CardHeader>
            <CardTitle>Sales Trend Custom Line Chart</CardTitle>
          </CardHeader>
          <CardContent>
            <LineChart queryKey="mocked_sales" parameters={salesParameters}>
              <CartesianGrid strokeDasharray="3 3" />
              <Line type="monotone" dataKey="revenue" stroke="#40d1f5" />
              <Line type="monotone" dataKey="expenses" stroke="#4462c9" />
              <Line type="monotone" dataKey="customers" stroke="#EB1600" />
              <XAxis dataKey="month" />
              <YAxis />
              <Tooltip />
            </LineChart>
          </CardContent>
        </Card>
        <Card className="shadow-lg flex min-w-0">
          <CardHeader>
            <CardTitle>Sales Trend Radar Chart</CardTitle>
          </CardHeader>
          <CardContent>
            <RadarChart queryKey="mocked_sales" parameters={salesParameters} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default App;
