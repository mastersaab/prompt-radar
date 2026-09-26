export interface RadarNode {
  id: string;
  prompt: string;
  category: string;
  x: number;
  y: number;
  hit_count?: number;
  tokens_input?: number;
  tokens_output?: number;
  response?: string;
}

export interface Coordinate {
  x: number;
  y: number;
}

export interface NeighborHit {
  id: string;
  prompt: string;
  category: string;
  similarity: number;
  x: number;
  y: number;
}

export interface QueryResult {
  cache_hit: boolean;
  similarity: number;
  threshold: number;
  latency_ms: number;
  baseline_ms: number;
  tokens_saved: number;
  cost_saved: number;
  query_coord: Coordinate;
  nearest_coord?: Coordinate;
  nearest_id?: string;
  nearest_prompt?: string;
  response: string;
  neighbors?: NeighborHit[];
  streaming?: boolean;
}

export interface ThresholdPoint {
  threshold: number;
  hit_rate_pct: number;
  projected_tokens_saved: number;
  projected_cost_saved_usd: number;
}

export interface AnalyticsSummary {
  total_requests: number;
  cache_hits: number;
  cache_misses: number;
  hit_rate_pct: number;
  total_tokens_saved: number;
  total_cost_saved_usd: number;
  total_time_saved_ms: number;
  avg_cache_latency_ms: number;
  avg_miss_latency_ms: number;
  threshold_curve: ThresholdPoint[];
}

export interface TelemetryLogItem {
  id?: number;
  queryId?: string;
  prompt: string;
  cacheHit: boolean;
  similarity: number;
  threshold: number;
  latencyMs: number;
  baselineMs: number;
  tokensSaved: number;
  costSaved: number;
  model: string;
  timestamp: string;
}
