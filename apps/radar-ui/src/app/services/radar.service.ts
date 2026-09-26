import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { AnalyticsSummary, RadarNode, TelemetryLogItem } from '../models/radar.models';

@Injectable({
  providedIn: 'root'
})
export class RadarService {
  private proxyUrl = 'http://localhost:8081';
  private controlPlaneUrl = 'http://localhost:8080';

  constructor(private http: HttpClient) {}

  getNodes(): Observable<{ total: number; nodes: RadarNode[] }> {
    return this.http.get<{ total: number; nodes: RadarNode[] }>(`${this.proxyUrl}/v1/nodes`).pipe(
      catchError(err => {
        console.warn('Failed to load nodes from proxy:', err);
        return of({ total: 0, nodes: [] });
      })
    );
  }

  getProxyHealth(): Observable<any> {
    return this.http.get(`${this.proxyUrl}/v1/health`).pipe(
      catchError(err => of({ status: 'OFFLINE', error: err.message }))
    );
  }

  getAnalyticsSummary(): Observable<AnalyticsSummary> {
    return this.http.get<AnalyticsSummary>(`${this.controlPlaneUrl}/api/v1/analytics/summary`).pipe(
      catchError(err => {
        console.warn('Control plane unreachable, using fallback summary:', err);
        return of({
          total_requests: 0,
          cache_hits: 0,
          cache_misses: 0,
          hit_rate_pct: 0,
          total_tokens_saved: 0,
          total_cost_saved_usd: 0,
          total_time_saved_ms: 0,
          avg_cache_latency_ms: 3.2,
          avg_miss_latency_ms: 1240.0,
          threshold_curve: [
            { threshold: 0.70, hit_rate_pct: 92.0, projected_tokens_saved: 14000, projected_cost_saved_usd: 0.0063 },
            { threshold: 0.75, hit_rate_pct: 88.0, projected_tokens_saved: 13200, projected_cost_saved_usd: 0.0059 },
            { threshold: 0.80, hit_rate_pct: 80.0, projected_tokens_saved: 12000, projected_cost_saved_usd: 0.0054 },
            { threshold: 0.85, hit_rate_pct: 64.0, projected_tokens_saved: 9600, projected_cost_saved_usd: 0.0043 },
            { threshold: 0.88, hit_rate_pct: 52.0, projected_tokens_saved: 7800, projected_cost_saved_usd: 0.0035 },
            { threshold: 0.90, hit_rate_pct: 44.0, projected_tokens_saved: 6600, projected_cost_saved_usd: 0.0030 },
            { threshold: 0.95, hit_rate_pct: 24.0, projected_tokens_saved: 3600, projected_cost_saved_usd: 0.0016 },
            { threshold: 0.98, hit_rate_pct: 15.0, projected_tokens_saved: 2250, projected_cost_saved_usd: 0.0010 }
          ]
        });
      })
    );
  }

  getRecentLogs(): Observable<TelemetryLogItem[]> {
    return this.http.get<TelemetryLogItem[]>(`${this.controlPlaneUrl}/api/v1/analytics/recent`).pipe(
      catchError(() => of([]))
    );
  }

  async streamQuery(
    prompt: string,
    threshold: number,
    onEvent: (event: any) => void
  ): Promise<void> {
    const response = await fetch(`${this.proxyUrl}/v1/query`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt, threshold, stream: true })
    });

    if (!response.ok) {
      throw new Error(`Proxy error: ${response.statusText}`);
    }

    const reader = response.body?.getReader();
    if (!reader) throw new Error('No readable stream available');

    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n\n');
      buffer = lines.pop() || '';

      for (const block of lines) {
        const line = block.trim();
        if (!line.startsWith('data: ')) continue;
        const dataStr = line.substring(6).trim();
        if (dataStr === '[DONE]') {
          onEvent({ type: 'done' });
          continue;
        }

        try {
          const parsed = JSON.parse(dataStr);
          onEvent(parsed);
        } catch (e) {
          console.error('Failed to parse SSE JSON chunk:', dataStr);
        }
      }
    }
  }
}
