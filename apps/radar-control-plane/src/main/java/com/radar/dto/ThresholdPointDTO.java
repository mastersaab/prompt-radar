package com.radar.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ThresholdPointDTO {

    @JsonProperty("threshold")
    private double threshold;

    @JsonProperty("hit_rate_pct")
    private double hitRatePct;

    @JsonProperty("projected_tokens_saved")
    private long projectedTokensSaved;

    @JsonProperty("projected_cost_saved_usd")
    private double projectedCostSavedUsd;

    public ThresholdPointDTO() {}

    public ThresholdPointDTO(double threshold, double hitRatePct, long projectedTokensSaved, double projectedCostSavedUsd) {
        this.threshold = threshold;
        this.hitRatePct = hitRatePct;
        this.projectedTokensSaved = projectedTokensSaved;
        this.projectedCostSavedUsd = projectedCostSavedUsd;
    }

    public double getThreshold() { return threshold; }
    public void setThreshold(double threshold) { this.threshold = threshold; }

    public double getHitRatePct() { return hitRatePct; }
    public void setHitRatePct(double hitRatePct) { this.hitRatePct = hitRatePct; }

    public long getProjectedTokensSaved() { return projectedTokensSaved; }
    public void setProjectedTokensSaved(long projectedTokensSaved) { this.projectedTokensSaved = projectedTokensSaved; }

    public double getProjectedCostSavedUsd() { return projectedCostSavedUsd; }
    public void setProjectedCostSavedUsd(double projectedCostSavedUsd) { this.projectedCostSavedUsd = projectedCostSavedUsd; }
}
