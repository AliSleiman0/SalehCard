package com.example.mobilebridgev2.dto

data class CommandDTO (
    val commandId: String,
    val provider: String,
    val commandType: String,
    val timestamp: Long,

    val recipientNumber: String?,
    val amount: Double?,
    val cardCode: String?,
    val message: String?
)
