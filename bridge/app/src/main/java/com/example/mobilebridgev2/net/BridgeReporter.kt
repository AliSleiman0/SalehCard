package com.example.mobilebridgev2.net

import android.util.Log
import com.example.mobilebridgev2.ProviderStore
import com.example.mobilebridgev2.dto.CommandResultDTO
import com.example.mobilebridgev2.dto.HeartbeatDTO
import com.example.mobilebridgev2.dto.ResultReportDTO
import com.example.mobilebridgev2.retrofit.RetrofitClient
import kotlinx.coroutines.delay

/**
 * All-HTTP replacement for the old WebSocket client. Reports command results to
 * the server (idempotently, with a bounded retry so a transient network blip
 * doesn't drop a completed recharge — the server dedupes duplicates), plus
 * heartbeat and diagnostic logs.
 */
object BridgeReporter {

    private const val TAG = "BridgeReporter"
    private const val APP_VERSION = "1.0"
    private const val MAX_RESULT_ATTEMPTS = 5

    /**
     * Reports a command result, retrying with backoff. A duplicate (the server
     * already recorded this command) counts as success. Returns true once the
     * server has the result.
     */
    suspend fun reportResult(result: CommandResultDTO): Boolean {
        val body = ResultReportDTO(
            statusCode = result.statusCode,
            transferredAmount = result.amount,
            billingAmount = result.billingAmount,
            balance = result.balance,
            validityDate = result.validityDate,
            rawReply = result.rawReply,
            errorMessage = result.errorMessage,
        )
        var attempt = 0
        while (attempt < MAX_RESULT_ATTEMPTS) {
            try {
                RetrofitClient.api.reportResult(result.id, body)
                return true
            } catch (e: Exception) {
                attempt++
                Log.w(TAG, "reportResult attempt $attempt failed for ${result.id}: ${e.message}")
                if (attempt < MAX_RESULT_ATTEMPTS) {
                    delay(1000L * attempt)
                }
            }
        }
        return false
    }

    /** Sends a liveness + SIM-state heartbeat (best effort). */
    suspend fun heartbeat() {
        try {
            RetrofitClient.api.heartbeat(
                HeartbeatDTO(
                    touchBalance = ProviderStore.touchSimBalance,
                    touchValidity = ProviderStore.touchSimValidityDate,
                    alfaBalance = ProviderStore.alfaSimBalance,
                    alfaValidity = ProviderStore.alfaSimValidityDate,
                    appVersion = APP_VERSION,
                )
            )
        } catch (e: Exception) {
            Log.w(TAG, "heartbeat failed: ${e.message}")
        }
    }

    /** Writes a diagnostic line to logcat (`adb logcat -s BridgeReporter`). Kept
     *  non-suspend so it's callable from any context; server-side log shipping
     *  (POST /logs) is available on the API for a future batched sender. */
    fun log(level: String, message: String) {
        Log.i(TAG, "[$level] $message")
    }
}
