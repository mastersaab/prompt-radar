import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RadarCanvasComponent } from './components/radar-canvas/radar-canvas.component';
import { PromptInputComponent } from './components/prompt-input/prompt-input.component';
import { StreamViewComponent } from './components/stream-view/stream-view.component';
import { MetricsBarComponent } from './components/metrics-bar/metrics-bar.component';
import { ThresholdCurveComponent } from './components/threshold-curve/threshold-curve.component';
import { TelemetryFeedComponent } from './components/telemetry-feed/telemetry-feed.component';
import { RadarService } from './services/radar.service';
import { AnalyticsSummary, Coordinate, QueryResult, RadarNode, TelemetryLogItem } from './models/radar.models';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule,
    RadarCanvasComponent,
    PromptInputComponent,
    StreamViewComponent,
    MetricsBarComponent,
    ThresholdCurveComponent,
    TelemetryFeedComponent
  ],
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit, OnDestroy {
  nodes: RadarNode[] = [];
  queryCoord: Coordinate | null = null;
  nearestCoord: Coordinate | null = null;
  threshold: number = 0.88;
  lastEvent: { type: 'hit' | 'miss_start' | 'miss_complete'; similarity?: number } | null = null;

  currentResult: QueryResult | null = null;
  streamText: string = '';
  isStreaming: boolean = false;

  analyticsSummary: AnalyticsSummary | null = null;
  telemetryLogs: TelemetryLogItem[] = [];

  proxyStatus: 'UP' | 'OFFLINE' | 'CONNECTING' = 'CONNECTING';
  controlPlaneStatus: 'UP' | 'OFFLINE' | 'CONNECTING' = 'CONNECTING';

  private pollInterval: any;

  constructor(private radarService: RadarService) {}

  ngOnInit() {
    this.refreshAllData();

    // Poll analytics every 4 seconds
    this.pollInterval = setInterval(() => {
      this.refreshAnalytics();
    }, 4000);
  }

  ngOnDestroy() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
    }
  }

  refreshAllData() {
    // 1. Fetch Proxy nodes
    this.radarService.getNodes().subscribe(res => {
      if (res && res.nodes) {
        this.nodes = res.nodes;
        this.proxyStatus = 'UP';
      }
    });

    // 2. Check Proxy health
    this.radarService.getProxyHealth().subscribe(res => {
      if (res.status === 'UP') {
        this.proxyStatus = 'UP';
      } else {
        this.proxyStatus = 'OFFLINE';
      }
    });

    // 3. Fetch Control Plane Analytics
    this.refreshAnalytics();
  }

  refreshAnalytics() {
    this.radarService.getAnalyticsSummary().subscribe(summary => {
      if (summary) {
        this.analyticsSummary = summary;
        this.controlPlaneStatus = 'UP';
      }
    });

    this.radarService.getRecentLogs().subscribe(logs => {
      if (logs && logs.length > 0) {
        this.telemetryLogs = logs;
      }
    });
  }

  async handleQuerySubmit(event: { prompt: string; threshold: number }) {
    if (this.isStreaming) return;

    this.threshold = event.threshold;
    this.isStreaming = true;
    this.streamText = '';
    this.currentResult = null;
    this.lastEvent = null;

    try {
      await this.radarService.streamQuery(event.prompt, event.threshold, (ev) => {
        if (ev.type === 'hit') {
          this.queryCoord = ev.query_coord;
          this.nearestCoord = ev.nearest_coord;
          this.lastEvent = { type: 'hit', similarity: ev.similarity };

          this.currentResult = {
            cache_hit: true,
            similarity: ev.similarity,
            threshold: ev.threshold,
            latency_ms: ev.latency_ms,
            baseline_ms: ev.baseline_ms,
            tokens_saved: ev.tokens_saved,
            cost_saved: ev.cost_saved,
            query_coord: ev.query_coord,
            nearest_coord: ev.nearest_coord,
            nearest_id: ev.nearest_id,
            nearest_prompt: ev.nearest_prompt,
            response: ev.response,
            neighbors: ev.neighbors,
            streaming: false
          };

          this.streamText = ev.response;
          this.isStreaming = false;

          // Add to local feed immediately for snappy feel
          this.appendLocalLog({
            prompt: event.prompt,
            cacheHit: true,
            similarity: ev.similarity,
            threshold: ev.threshold,
            latencyMs: ev.latency_ms,
            baselineMs: ev.baseline_ms,
            tokensSaved: ev.tokens_saved,
            costSaved: ev.cost_saved,
            model: 'semantic-cache-768d',
            timestamp: new Date().toISOString()
          });

        } else if (ev.type === 'miss_start') {
          this.queryCoord = ev.query_coord;
          this.nearestCoord = ev.nearest_coord;
          this.lastEvent = { type: 'miss_start', similarity: ev.similarity };

          this.currentResult = {
            cache_hit: false,
            similarity: ev.similarity,
            threshold: ev.threshold,
            latency_ms: 0,
            baseline_ms: 1250,
            tokens_saved: 0,
            cost_saved: 0,
            query_coord: ev.query_coord,
            nearest_coord: ev.nearest_coord,
            nearest_id: ev.nearest_id,
            nearest_prompt: ev.nearest_prompt,
            response: '',
            neighbors: ev.neighbors,
            streaming: true
          };

        } else if (ev.type === 'token') {
          this.streamText += ev.text;
          if (this.currentResult) {
            this.currentResult.response = this.streamText;
          }

        } else if (ev.type === 'miss_complete') {
          if (this.currentResult) {
            this.currentResult.streaming = false;
            this.currentResult.latency_ms = ev.latency_ms;
          }
          this.isStreaming = false;
          this.lastEvent = { type: 'miss_complete' };

          // Add novel query node to canvas nodes list!
          if (this.queryCoord) {
            this.nodes = [...this.nodes, {
              id: ev.new_entry_id || 'cache-new',
              prompt: event.prompt,
              category: 'General',
              x: this.queryCoord.x,
              y: this.queryCoord.y,
              hit_count: 0
            }];
          }

          this.appendLocalLog({
            prompt: event.prompt,
            cacheHit: false,
            similarity: this.currentResult?.similarity || 0,
            threshold: event.threshold,
            latencyMs: ev.latency_ms,
            baselineMs: 1250,
            tokensSaved: 0,
            costSaved: 0,
            model: 'gemini-1.5-flash',
            timestamp: new Date().toISOString()
          });

          // Trigger analytics refresh
          setTimeout(() => this.refreshAnalytics(), 500);

        } else if (ev.type === 'done') {
          this.isStreaming = false;
        }
      });
    } catch (err: any) {
      console.error('Query streaming error:', err);
      this.isStreaming = false;
    }
  }

  handleThresholdChange(newThreshold: number) {
    this.threshold = newThreshold;
  }

  private appendLocalLog(item: TelemetryLogItem) {
    this.telemetryLogs = [item, ...this.telemetryLogs.slice(0, 19)];
  }
}
