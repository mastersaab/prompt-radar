package com.radar.controller;

import com.radar.dto.TelemetryEventDTO;
import com.radar.service.AnalyticsService;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Collections;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/v1/telemetry")
public class TelemetryController {

    private final AnalyticsService analyticsService;

    public TelemetryController(AnalyticsService analyticsService) {
        this.analyticsService = analyticsService;
    }

    @PostMapping
    public ResponseEntity<Map<String, Object>> ingestEvents(@RequestBody List<TelemetryEventDTO> events) {
        analyticsService.ingestBatch(events);
        return ResponseEntity.status(HttpStatus.ACCEPTED).body(Map.of(
                "status", "INGESTED",
                "count", events != null ? events.size() : 0
        ));
    }
}
