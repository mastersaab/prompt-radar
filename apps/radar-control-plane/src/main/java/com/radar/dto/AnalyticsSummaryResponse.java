package com.radar.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class AnalyticsSummaryResponse {

    @JsonProperty("total_requests")
    private long totalRequests;

    @JsonProperty("cache_hits")
    private long cacheHits;

    @JsonProperty("cache_misses")
    private long cacheMisses;

    @JsonProperty("hit_rate_pct")
    private double hitRatePct;

    @JsonProperty("total_tokens_saved")
    private long totalTokensSaved;

    @JsonProperty("total_cost_saved_usd")
    private double totalCostSavedUsd;

    @JsonProperty("total_time_saved_ms")
    private long totalTimeSavedMs;

    @JsonProperty("avg_cache_latency_ms")
    private double avgCacheLatencyMs;

    @JsonProperty("avg_miss_latency_ms")
    private double avgMissLatencyMs;

    @JsonProperty("threshold_curve")
    private List<ThresholdPointDTO> thresholdCurve;

    public AnalyticsSummaryResponse() {}

    // Getters and Setters
    public long getTotalRequests() { return totalRequests; }
    public void setTotalRequests(long totalRequests) { this.totalRequests = totalRequests; }

    public long getCacheHits() { return cacheHits; }
    public void setCacheHits(long cacheHits) { this.cacheHits = cacheHits; }

    public long getCacheMisses() { return cacheMisses; }
    public void setCacheMisses(long cacheMisses) { this.cacheMisses = cacheMisses; }

    public double getHitRatePct() { return hitRatePct; }
    public void setHitRatePct(double hitRatePct) { this.hitRatePct = hitRatePct; }

    public long getTotalTokensSaved() { return totalTokensSaved; }
    public void setTotalTokensSaved(long totalTokensSaved) { this.totalTokensSaved = totalTokensSaved; }

    public double getTotalCostSavedUsd() { return totalCostSavedUsd; }
    public void setTotalCostSavedUsd(double totalCostSavedUsd) { this.totalCostSavedUsd = totalCostSavedUsd; }

    public long getTotalTimeSavedMs() { return totalTimeSavedMs; }
    public void setTotalTimeSavedMs(long totalTimeSavedMs) { this.totalTimeSavedMs = totalTimeSavedMs; }

    public double getAvgCacheLatencyMs() { return avgCacheLatencyMs; }
    public void setAvgCacheLatencyMs(double avgCacheLatencyMs) { this.avgCacheLatencyMs = avgCacheLatencyMs; }

    public double getAvgMissLatencyMs() { return avgMissLatencyMs; }
    public void setAvgMissLatencyMs(double avgMissLatencyMs) { this.avgMissLatencyMs = avgMissLatencyMs; }

    public List<ThresholdPointDTO> getThresholdCurve() { return thresholdCurve; }
    public void setThresholdCurve(List<ThresholdPointDTO> thresholdCurve) { this.thresholdCurve = thresholdCurve; }
}
