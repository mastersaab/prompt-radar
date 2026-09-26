package com.radar.repository;

import com.radar.domain.TelemetryLog;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public interface TelemetryLogRepository extends JpaRepository<TelemetryLog, Long> {

    @Query("SELECT COUNT(t) FROM TelemetryLog t")
    long countTotalRequests();

    @Query("SELECT COUNT(t) FROM TelemetryLog t WHERE t.cacheHit = true")
    long countCacheHits();

    @Query("SELECT COALESCE(SUM(t.tokensSaved), 0) FROM TelemetryLog t")
    long sumTokensSaved();

    @Query("SELECT COALESCE(SUM(t.costSaved), 0.0) FROM TelemetryLog t")
    double sumCostSaved();

    @Query("SELECT COALESCE(AVG(t.latencyMs), 0.0) FROM TelemetryLog t WHERE t.cacheHit = true")
    double averageCacheLatencyMs();

    @Query("SELECT COALESCE(AVG(t.latencyMs), 0.0) FROM TelemetryLog t WHERE t.cacheHit = false")
    double averageMissLatencyMs();

    @Query("SELECT COALESCE(SUM(t.baselineMs - t.latencyMs), 0) FROM TelemetryLog t WHERE t.cacheHit = true")
    long totalTimeSavedMs();

    @Query("SELECT t FROM TelemetryLog t ORDER BY t.timestamp DESC")
    List<TelemetryLog> findRecentLogs(Pageable pageable);
}
