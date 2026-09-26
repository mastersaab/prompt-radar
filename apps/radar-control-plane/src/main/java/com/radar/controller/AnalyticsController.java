package com.radar.controller;

import com.radar.domain.TelemetryLog;
import com.radar.dto.AnalyticsSummaryResponse;
import com.radar.dto.ThresholdPointDTO;
import com.radar.service.AnalyticsService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/v1/analytics")
public class AnalyticsController {

    private final AnalyticsService analyticsService;

    public AnalyticsController(AnalyticsService analyticsService) {
        this.analyticsService = analyticsService;
    }

    @GetMapping("/summary")
    public ResponseEntity<AnalyticsSummaryResponse> getSummary() {
        return ResponseEntity.ok(analyticsService.getSummary());
    }

    @GetMapping("/recent")
    public ResponseEntity<List<TelemetryLog>> getRecent(@RequestParam(defaultValue = "20") int limit) {
        return ResponseEntity.ok(analyticsService.getRecent(limit));
    }
}
