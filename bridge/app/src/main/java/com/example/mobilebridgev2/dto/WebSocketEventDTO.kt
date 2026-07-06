package com.example.mobilebridgev2.dto

data class WebSocketEventDTO(
    val type: String,
    val provider: String? = null,
    val message: String? = null,
    val amount: Double? = null,
    val validityDays: Int? = null,
    val configuration: ConfigurationDTO? = null
)