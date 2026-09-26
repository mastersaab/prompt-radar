package com.radar.domain;

import jakarta.persistence.*;
import java.time.Instant;

@Entity
@Table(name = "telemetry_logs", indexes = {
    @Index(name = "idx_telemetry_created_at", columnList = "timestamp"),
    @Index(name = "idx_telemetry_cache_hit", columnList = "cacheHit")
})
public class TelemetryLog {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "query_id", length = 64)
    private String queryId;

    @Column(name = "prompt", columnDefinition = "TEXT")
    private String prompt;

    @Column(name = "cache_hit")
    private boolean cacheHit;

    @Column(name = "similarity")
    private Float similarity;

    @Column(name = "threshold")
    private Float threshold;

    @Column(name = "latency_ms")
    private Long latencyMs;

    @Column(name = "baseline_ms")
    private Long baselineMs;

    @Column(name = "tokens_input")
    private Integer tokensInput;

    @Column(name = "tokens_output")
    private Integer tokensOutput;

    @Column(name = "tokens_saved")
    private Integer tokensSaved;

    @Column(name = "cost_saved")
    private Double costSaved;

    @Column(name = "model", length = 64)
    private String model;

    @Column(name = "timestamp")
    private Instant timestamp;

    public TelemetryLog() {}

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public String getQueryId() { return queryId; }
    public void setQueryId(String queryId) { this.queryId = queryId; }

    public String getPrompt() { return prompt; }
    public void setPrompt(String prompt) { this.prompt = prompt; }

    public boolean isCacheHit() { return cacheHit; }
    public void setCacheHit(boolean cacheHit) { this.cacheHit = cacheHit; }

    public Float getSimilarity() { return similarity; }
    public void setSimilarity(Float similarity) { this.similarity = similarity; }

    public Float getThreshold() { return threshold; }
    public void setThreshold(Float threshold) { this.threshold = threshold; }

    public Long getLatencyMs() { return latencyMs; }
    public void setLatencyMs(Long latencyMs) { this.latencyMs = latencyMs; }

    public Long getBaselineMs() { return baselineMs; }
    public void setBaselineMs(Long baselineMs) { this.baselineMs = baselineMs; }

    public Integer getTokensInput() { return tokensInput; }
    public void setTokensInput(Integer tokensInput) { this.tokensInput = tokensInput; }

    public Integer getTokensOutput() { return tokensOutput; }
    public void setTokensOutput(Integer tokensOutput) { this.tokensOutput = tokensOutput; }

    public Integer getTokensSaved() { return tokensSaved; }
    public void setTokensSaved(Integer tokensSaved) { this.tokensSaved = tokensSaved; }

    public Double getCostSaved() { return costSaved; }
    public void setCostSaved(Double costSaved) { this.costSaved = costSaved; }

    public String getModel() { return model; }
    public void setModel(String model) { this.model = model; }

    public Instant getTimestamp() { return timestamp; }
    public void setTimestamp(Instant timestamp) { this.timestamp = timestamp; }
}
