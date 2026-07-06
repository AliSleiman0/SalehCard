package com.example.mobilebridgev2.dto

/** The command result the device POSTs to /commands/{id}/result. Matches the
 *  Go bridge resultDTO field-for-field (camelCase, Gson identity naming). */
data class ResultReportDTO(
    val statusCode: Int,
    val transferredAmount: Double? = null,
    val billingAmount: Double? = null,
    val balance: Double? = null,
    val validityDate: String? = null,
    val rawReply: String? = null,
    val errorMessage: String? = null,
)

/** The server's ack for a reported result: {"status":"recorded"|"duplicate"}. */
data class ResultAckDTO(
    val status: String = "",
)

/** The device's periodic liveness + SIM-state report (POST /heartbeat). */
data class HeartbeatDTO(
    val touchBalance: Double? = null,
    val touchValidity: String? = null,
    val alfaBalance: Double? = null,
    val alfaValidity: String? = null,
    val appVersion: String? = null,
)

/** A batch of device diagnostic logs (POST /logs). */
data class LogsDTO(
    val entries: List<LogItemDTO>,
)

data class LogItemDTO(
    val level: String,
    val errorCode: Int? = null,
    val message: String,
)
