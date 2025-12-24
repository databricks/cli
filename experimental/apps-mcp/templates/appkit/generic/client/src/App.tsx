import {
  useAnalyticsQuery,
  AreaChart,
  LineChart,
  RadarChart,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Skeleton,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@databricks/appkit-ui/react';
import { sql } from "@databricks/appkit-ui/js";
import { useState, useEffect } from 'react';

function App() {
  const { data, loading, error } = useAnalyticsQuery('hello_world', {
    message: sql.string('hello world'),
  });

  const [health, setHealth] = useState<{
    status: string;
    timestamp: string;
  } | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/health')
      .then((response) => response.json())
      .then((data: { status: string }) => setHealth({ ...data, timestamp: new Date().toISOString() }))
      .catch((error: { message: string }) => setHealthError(error.message));
  }, []);

  const [maxMonthNum, setMaxMonthNum] = useState<number>(12);

  const salesParameters = { max_month_num: sql.number(maxMonthNum) };

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

        <Card className="shadow-lg md:col-span-2">
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
                  <div className="w-2 h-2 rounded-full bg-success animate-pulse"></div>
                  <div className="text-lg font-semibold text-success">{health.status.toUpperCase()}</div>
                </div>
                <div className="text-sm text-muted-foreground">
                  Last checked: {new Date(health.timestamp).toLocaleString()}
                </div>
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
                    {Array.from({ length: 12 }, (_, i) => (
                      <SelectItem key={`month-${i + 1}`} value={(i + 1).toString()}>
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
            <LineChart queryKey="mocked_sales" parameters={salesParameters} />
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
