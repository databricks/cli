import { useAnalyticsQuery } from '@databricks/app-kit/react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { QueryResult } from '../../shared/types';
import { trpc } from './lib/trpc';
import { useState, useEffect } from 'react';

function App() {
  const { data, loading, error } = useAnalyticsQuery<QueryResult[]>('hello_world', {});

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

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800 flex flex-col items-center justify-center p-4">
      <div className="mb-8 text-center">
        <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-slate-900 to-slate-700 dark:from-slate-100 dark:to-slate-300 bg-clip-text text-transparent">
          Minimal Databricks App
        </h1>
        <p className="text-lg text-muted-foreground max-w-md">
          A minimal Databricks App powered by the Databricks Apps SDK
        </p>
      </div>

      <div className="flex flex-col gap-6 w-full max-w-md">
        <Card className="shadow-lg">
          <CardHeader>
            <CardTitle>SQL Query Result</CardTitle>
          </CardHeader>
          <CardContent>
            {loading && <div className="text-muted-foreground animate-pulse">Loading query results...</div>}
            {error && <div className="text-destructive bg-destructive/10 p-3 rounded-md">Error: {error}</div>}
            {data && data.length > 0 && (
              <div className="space-y-2">
                <div className="text-sm text-muted-foreground">Query: SELECT &apos;hello world&apos; AS value</div>
                <div className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                  {data[0].value}
                </div>
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
              <div className="text-muted-foreground animate-pulse">Checking server health...</div>
            )}
            {healthError && (
              <div className="text-destructive bg-destructive/10 p-3 rounded-md">Error: {healthError}</div>
            )}
            {health && (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
                  <div className="text-lg font-semibold text-green-600 dark:text-green-400">
                    {health.status.toUpperCase()}
                  </div>
                </div>
                <div className="text-sm text-muted-foreground">
                  Last checked: {new Date(health.timestamp).toLocaleString()}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default App;
