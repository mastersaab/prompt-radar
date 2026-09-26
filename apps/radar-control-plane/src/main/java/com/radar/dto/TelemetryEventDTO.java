package com.radar.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;

public class TelemetryEventDTO {

    @JsonProperty("query_id")
    private String queryId;

    @JsonProperty("prompt")
    private String prompt;

    @JsonProperty("cache_hit")
    private boolean cacheHit;

    @JsonProperty("similarity")
    private Float similarity;

    @JsonProperty("threshold")
    private Float threshold;

    @JsonProperty("latency_ms")
    private Long latencyMs;

    @JsonProperty("baseline_ms")
    private Long baselineMs;

    @JsonProperty("tokens_input")
    private Integer tokensInput;

    @JsonProperty("tokens_output")
    private Integer tokensOutput;

    @JsonProperty("tokens_saved")
    private Integer tokensSaved;

    @JsonProperty("cost_saved")
    private Double costSaved;

    @JsonProperty("model")
    private String model;

    @JsonProperty("timestamp")
    private Instant timestamp;

    public TelemetryEventDTO() {}

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
