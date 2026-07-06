package com.example.mobilebridgev2.retrofit

import com.example.mobilebridgev2.dto.CommandDTO
import com.example.mobilebridgev2.dto.ConfigurationDTO
import com.example.mobilebridgev2.dto.HeartbeatDTO
import com.example.mobilebridgev2.dto.LogsDTO
import com.example.mobilebridgev2.dto.ResultAckDTO
import com.example.mobilebridgev2.dto.ResultReportDTO
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * The device HTTP contract (base = <server>/api/v1/bridge/). Authentication is
 * the per-device bearer token, added by RetrofitClient's interceptor — the
 * device is identified by its token, not a query param. All-HTTP: results are
 * POSTed (idempotently, with retry), replacing the old WebSocket.
 */
interface BridgeApi {

    @GET("config")
    suspend fun getConfig(): ConfigurationDTO

    /** Leases up to [max] queued commands for this device's operators. */
    @GET("commands")
    suspend fun getCommands(@Query("max") max: Int): List<CommandDTO>

    /** Reports a command result. Idempotent server-side (duplicate → 200). */
    @POST("commands/{id}/result")
    suspend fun reportResult(@Path("id") commandId: String, @Body body: ResultReportDTO): ResultAckDTO

    @POST("heartbeat")
    suspend fun heartbeat(@Body body: HeartbeatDTO): Response<Unit>

    @POST("logs")
    suspend fun postLogs(@Body body: LogsDTO): Response<Unit>
}
